package main

/*
#cgo pkg-config: vips
#cgo LDFLAGS: -s -w
#include "vips.h"
*/
import "C"

import (
	"errors"
	"log"
	"math"
	"os"
	"runtime"
	"unsafe"
)

type imageType int

const (
	UNKNOWN = C.UNKNOWN
	JPEG    = C.JPEG
	PNG     = C.PNG
	WEBP    = C.WEBP
	GIF     = C.GIF
)

var imageTypes = map[string]imageType{
	"jpeg": JPEG,
	"jpg":  JPEG,
	"png":  PNG,
	"webp": WEBP,
	"gif":  GIF,
}

type gravityType int

const (
	CENTER gravityType = iota
	NORTH
	EAST
	SOUTH
	WEST
	SMART
)

var gravityTypes = map[string]gravityType{
	"ce": CENTER,
	"no": NORTH,
	"ea": EAST,
	"so": SOUTH,
	"we": WEST,
	"sm": SMART,
}

type resizeType int

const (
	FIT resizeType = iota
	FILL
	CROP
)

var resizeTypes = map[string]resizeType{
	"fit":  FIT,
	"fill": FILL,
	"crop": CROP,
}

type processingOptions struct {
	Resize  resizeType
	Width   int
	Height  int
	Gravity gravityType
	Enlarge bool
	Format  imageType
}

var vipsSupportSmartcrop bool
var vipsTypeSupportLoad = make(map[imageType]bool)
var vipsTypeSupportSave = make(map[imageType]bool)

type cConfig struct {
	Quality         C.int
	JpegProgressive C.int
	PngInterlaced   C.int
}

var cConf cConfig

func initVips() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := C.vips_initialize(); err != 0 {
		C.vips_shutdown()
		log.Fatalln("unable to start vips!")
	}

	C.vips_cache_set_max_mem(100 * 1024 * 1024) // 100Mb
	C.vips_cache_set_max(500)

	if len(os.Getenv("IMGPROXY_VIPS_LEAK_CHECK")) > 0 {
		C.vips_leak_set(C.gboolean(1))
	}

	if len(os.Getenv("IMGPROXY_VIPS_CACHE_TRACE")) > 0 {
		C.vips_cache_set_trace(C.gboolean(1))
	}

	vipsSupportSmartcrop = C.vips_support_smartcrop() == 1

	if int(C.vips_type_find_load_go(C.JPEG)) != 0 {
		vipsTypeSupportLoad[JPEG] = true
	}
	if int(C.vips_type_find_load_go(C.PNG)) != 0 {
		vipsTypeSupportLoad[PNG] = true
	}
	if int(C.vips_type_find_load_go(C.WEBP)) != 0 {
		vipsTypeSupportLoad[WEBP] = true
	}
	if int(C.vips_type_find_load_go(C.GIF)) != 0 {
		vipsTypeSupportLoad[GIF] = true
	}

	if int(C.vips_type_find_save_go(C.JPEG)) != 0 {
		vipsTypeSupportSave[JPEG] = true
	}
	if int(C.vips_type_find_save_go(C.PNG)) != 0 {
		vipsTypeSupportSave[PNG] = true
	}
	if int(C.vips_type_find_save_go(C.WEBP)) != 0 {
		vipsTypeSupportSave[WEBP] = true
	}

	cConf.Quality = C.int(conf.Quality)

	if conf.JpegProgressive {
		cConf.JpegProgressive = C.int(1)
	}

	if conf.PngInterlaced {
		cConf.PngInterlaced = C.int(1)
	}
}

func shutdownVips() {
	C.vips_shutdown()
}

func randomAccessRequired(po processingOptions) int {
	if po.Gravity == SMART {
		return 1
	}
	return 0
}

func round(f float64) int {
	return int(f + .5)
}

func extractMeta(img *C.VipsImage) (int, int, int, bool) {
	width := int(img.Xsize)
	height := int(img.Ysize)

	angle := C.VIPS_ANGLE_D0
	flip := false

	orientation := C.vips_get_exif_orientation(img)
	if orientation >= 5 && orientation <= 8 {
		width, height = height, width
	}
	if orientation == 3 || orientation == 4 {
		angle = C.VIPS_ANGLE_D180
	}
	if orientation == 5 || orientation == 6 {
		angle = C.VIPS_ANGLE_D90
	}
	if orientation == 7 || orientation == 8 {
		angle = C.VIPS_ANGLE_D270
	}
	if orientation == 2 || orientation == 4 || orientation == 5 || orientation == 7 {
		flip = true
	}

	return width, height, angle, flip
}

func calcScale(width, height int, po processingOptions) float64 {
	if (po.Width == width && po.Height == height) || (po.Resize != FILL && po.Resize != FIT) {
		return 1
	}

	fsw, fsh, fow, foh := float64(width), float64(height), float64(po.Width), float64(po.Height)

	wr := fow / fsw
	hr := foh / fsh

	if po.Resize == FIT {
		return math.Min(wr, hr)
	}

	return math.Max(wr, hr)
}

func calcShink(scale float64, imgtype imageType) int {
	shrink := int(1.0 / scale)

	if imgtype != JPEG {
		return shrink
	}

	switch {
	case shrink >= 16:
		return 8
	case shrink >= 8:
		return 4
	case shrink >= 4:
		return 2
	}

	return 1
}

func calcCrop(width, height int, po processingOptions) (left, top int) {
	left = (width - po.Width + 1) / 2
	top = (height - po.Height + 1) / 2

	if po.Gravity == NORTH {
		top = 0
	}

	if po.Gravity == EAST {
		left = width - po.Width
	}

	if po.Gravity == SOUTH {
		top = height - po.Height
	}

	if po.Gravity == WEST {
		left = 0
	}

	return
}

func processImage(data []byte, imgtype imageType, po processingOptions, t *timer) ([]byte, error) {
	defer C.vips_cleanup()
	defer runtime.KeepAlive(data)

	if po.Gravity == SMART && !vipsSupportSmartcrop {
		return nil, errors.New("Smart crop is not supported by used version of libvips")
	}

	img, err := vipsLoadImage(data, imgtype, 1)
	if err != nil {
		return nil, err
	}
	defer C.clear_image(&img)

	t.Check()

	imgWidth, imgHeight, angle, flip := extractMeta(img)

	// Ensure we won't crop out of bounds
	if !po.Enlarge || po.Resize == CROP {
		if imgWidth < po.Width {
			po.Width = imgWidth
		}

		if imgHeight < po.Height {
			po.Height = imgHeight
		}
	}

	if po.Width != imgWidth || po.Height != imgHeight {
		if po.Resize == FILL || po.Resize == FIT {
			scale := calcScale(imgWidth, imgHeight, po)

			// Do some shrink-on-load
			if scale < 1.0 {
				if imgtype == JPEG || imgtype == WEBP {
					shrink := calcShink(scale, imgtype)
					scale = scale * float64(shrink)

					if tmp, e := vipsLoadImage(data, imgtype, shrink); e == nil {
						C.swap_and_clear(&img, tmp)
					} else {
						return nil, e
					}
				}
			}

			premultiplied := false
			var bandFormat C.VipsBandFormat

			if vipsImageHasAlpha(img) {
				if bandFormat, err = vipsPremultiply(&img); err != nil {
					return nil, err
				}
				premultiplied = true
			}

			if err = vipsResize(&img, scale); err != nil {
				return nil, err
			}

			if premultiplied {
				if err = vipsUnpremultiply(&img, bandFormat); err != nil {
					return nil, err
				}
			}
		}
	}

	if err = vipsImportColourProfile(&img); err != nil {
		return nil, err
	}

	if err = vipsFixColourspace(&img); err != nil {
		return nil, err
	}

	t.Check()

	if angle != C.VIPS_ANGLE_D0 || flip {
		if err = vipsImageCopyMemory(&img); err != nil {
			return nil, err
		}

		if angle != C.VIPS_ANGLE_D0 {
			if err = vipsRotate(&img, angle); err != nil {
				return nil, err
			}
		}

		if flip {
			if err = vipsFlip(&img); err != nil {
				return nil, err
			}
		}
	}

	t.Check()

	if (po.Width != imgWidth || po.Height != imgHeight) && (po.Resize == FILL || po.Resize == CROP) {
		if po.Gravity == SMART {
			if err = vipsImageCopyMemory(&img); err != nil {
				return nil, err
			}
			if err = vipsSmartCrop(&img, po.Width, po.Height); err != nil {
				return nil, err
			}
		} else {
			left, top := calcCrop(int(img.Xsize), int(img.Ysize), po)
			if err = vipsCrop(&img, left, top, po.Width, po.Height); err != nil {
				return nil, err
			}
		}

		t.Check()
	}

	return vipsSaveImage(img, po.Format)
}

func vipsLoadImage(data []byte, imgtype imageType, shrink int) (*C.struct__VipsImage, error) {
	var img *C.struct__VipsImage
	if C.vips_load_buffer(unsafe.Pointer(&data[0]), C.size_t(len(data)), C.int(imgtype), C.int(shrink), &img) != 0 {
		return nil, vipsError()
	}
	return img, nil
}

func vipsSaveImage(img *C.struct__VipsImage, imgtype imageType) ([]byte, error) {
	var ptr unsafe.Pointer
	defer C.g_free_go(&ptr)

	err := C.int(0)

	imgsize := C.size_t(0)

	switch imgtype {
	case JPEG:
		err = C.vips_jpegsave_go(img, &ptr, &imgsize, 1, cConf.Quality, cConf.JpegProgressive)
	case PNG:
		err = C.vips_pngsave_go(img, &ptr, &imgsize, cConf.PngInterlaced)
	case WEBP:
		err = C.vips_webpsave_go(img, &ptr, &imgsize, 1, cConf.Quality)
	}
	if err != 0 {
		return nil, vipsError()
	}

	return C.GoBytes(ptr, C.int(imgsize)), nil
}

func vipsImageHasAlpha(img *C.struct__VipsImage) bool {
	return C.vips_image_hasalpha_go(img) > 0
}

func vipsPremultiply(img **C.struct__VipsImage) (C.VipsBandFormat, error) {
	var tmp *C.struct__VipsImage

	format := C.vips_band_format(*img)

	if C.vips_premultiply_go(*img, &tmp) != 0 {
		return 0, vipsError()
	}

	C.swap_and_clear(img, tmp)
	return format, nil
}

func vipsUnpremultiply(img **C.struct__VipsImage, format C.VipsBandFormat) error {
	var tmp *C.struct__VipsImage

	if C.vips_unpremultiply_go(*img, &tmp) != 0 {
		return vipsError()
	}
	C.swap_and_clear(img, tmp)

	if C.vips_cast_go(*img, &tmp, format) != 0 {
		return vipsError()
	}
	C.swap_and_clear(img, tmp)

	return nil
}

func vipsResize(img **C.struct__VipsImage, scale float64) error {
	var tmp *C.struct__VipsImage

	if C.vips_resize_go(*img, &tmp, C.double(scale)) != 0 {
		return vipsError()
	}

	C.swap_and_clear(img, tmp)
	return nil
}

func vipsRotate(img **C.struct__VipsImage, angle int) error {
	var tmp *C.struct__VipsImage

	if C.vips_rot_go(*img, &tmp, C.VipsAngle(angle)) != 0 {
		return vipsError()
	}

	C.swap_and_clear(img, tmp)
	return nil
}

func vipsFlip(img **C.struct__VipsImage) error {
	var tmp *C.struct__VipsImage

	if C.vips_flip_horizontal_go(*img, &tmp) != 0 {
		return vipsError()
	}

	C.swap_and_clear(img, tmp)
	return nil
}

func vipsCrop(img **C.struct__VipsImage, left, top, width, height int) error {
	var tmp *C.struct__VipsImage

	if C.vips_extract_area_go(*img, &tmp, C.int(left), C.int(top), C.int(width), C.int(height)) != 0 {
		return vipsError()
	}

	C.swap_and_clear(img, tmp)
	return nil
}

func vipsSmartCrop(img **C.struct__VipsImage, width, height int) error {
	var tmp *C.struct__VipsImage

	if C.vips_smartcrop_go(*img, &tmp, C.int(width), C.int(height)) != 0 {
		return vipsError()
	}

	C.swap_and_clear(img, tmp)
	return nil
}

func vipsImportColourProfile(img **C.struct__VipsImage) error {
	var tmp *C.struct__VipsImage

	if C.vips_need_icc_import(*img) > 0 {
		profile, err := cmykProfilePath()
		if err != nil {
			return err
		}

		if C.vips_icc_import_go(*img, &tmp, C.CString(profile)) != 0 {
			return vipsError()
		}
		C.swap_and_clear(img, tmp)
	}

	return nil
}

func vipsFixColourspace(img **C.struct__VipsImage) error {
	var tmp *C.struct__VipsImage

	if C.vips_image_guess_interpretation(*img) != C.VIPS_INTERPRETATION_sRGB {
		if C.vips_colourspace_go(*img, &tmp, C.VIPS_INTERPRETATION_sRGB) != 0 {
			return vipsError()
		}
		C.swap_and_clear(img, tmp)
	}

	return nil
}

func vipsImageCopyMemory(img **C.struct__VipsImage) error {
	var tmp *C.struct__VipsImage
	if tmp = C.vips_image_copy_memory(*img); tmp == nil {
		return vipsError()
	}
	C.swap_and_clear(img, tmp)
	return nil
}

func vipsError() error {
	return errors.New(C.GoString(C.vips_error_buffer()))
}
