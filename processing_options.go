package main

import (
	"C"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

func defaultProcessingOptions() processingOptions {
	return processingOptions{
		Resize:  FIT,
		Width:   0,
		Height:  0,
		Gravity: CENTER,
		Enlarge: false,
		Format:  JPEG,
	}
}

func decodeUrl(parts []string) (string, imageType, error) {
	var imgType imageType = JPEG

	urlParts := strings.Split(strings.Join(parts, ""), ".")

	if len(urlParts) > 2 {
		return "", 0, errors.New("Invalid url encoding")
	}

	if len(urlParts) == 2 {
		if f, ok := imageTypes[urlParts[1]]; ok {
			imgType = f
		} else {
			return "", 0, fmt.Errorf("Invalid image format: %s", urlParts[1])
		}
	}

	url, err := base64.RawURLEncoding.DecodeString(urlParts[0])
	if err != nil {
		return "", 0, errors.New("Invalid url encoding")
	}

	return string(url), imgType, nil
}

func applyWidthOption(po *processingOptions, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("Invalid width arguments: %v", args)
	}

	if w, err := strconv.Atoi(args[0]); err == nil || w >= 0 {
		po.Width = w
	} else {
		return fmt.Errorf("Invalid width: %s", args[0])
	}

	return nil
}

func applyHeightOption(po *processingOptions, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("Invalid height arguments: %v", args)
	}

	if h, err := strconv.Atoi(args[0]); err == nil || po.Height >= 0 {
		po.Height = h
	} else {
		return fmt.Errorf("Invalid height: %s", args[0])
	}

	return nil
}

func applyEnlargeOption(po *processingOptions, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("Invalid enlarge arguments: %v", args)
	}

	po.Enlarge = args[0] != "0"

	return nil
}

func applySizeOption(po *processingOptions, args []string) (err error) {
	if len(args) > 3 {
		return fmt.Errorf("Invalid size arguments: %v", args)
	}

	if len(args) >= 1 {
		if err = applyWidthOption(po, args[0:1]); err != nil {
			return
		}
	}

	if len(args) >= 2 {
		if err = applyHeightOption(po, args[1:2]); err != nil {
			return
		}
	}

	if len(args) == 3 {
		if err = applyEnlargeOption(po, args[2:3]); err != nil {
			return
		}
	}

	return nil
}

func applyResizeOption(po *processingOptions, args []string) error {
	if len(args) > 4 {
		return fmt.Errorf("Invalid resize arguments: %v", args)
	}

	if r, ok := resizeTypes[args[0]]; ok {
		po.Resize = r
	} else {
		return fmt.Errorf("Invalid resize type: %s", args[0])
	}

	if len(args) > 1 {
		if err := applySizeOption(po, args[1:]); err != nil {
			return err
		}
	}

	return nil
}

func applyGravityOption(po *processingOptions, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("Invalid resize arguments: %v", args)
	}

	if g, ok := gravityTypes[args[0]]; ok {
		po.Gravity = g
	} else {
		return fmt.Errorf("Invalid gravity: %s", args[0])
	}

	return nil
}

func applyFormatOption(po *processingOptions, imgType imageType) error {
	if !vipsTypeSupportSave[imgType] {
		return errors.New("Resulting image type not supported")
	}

	po.Format = imgType

	return nil
}

func applyProcessingOption(po *processingOptions, name string, args []string) error {
	switch name {
	case "resize":
		if err := applyResizeOption(po, args); err != nil {
			return err
		}
	case "size":
		if err := applySizeOption(po, args); err != nil {
			return err
		}
	case "width":
		if err := applyWidthOption(po, args); err != nil {
			return err
		}
	case "height":
		if err := applyHeightOption(po, args); err != nil {
			return err
		}
	case "enlarge":
		if err := applyEnlargeOption(po, args); err != nil {
			return err
		}
	case "gravity":
		if err := applyGravityOption(po, args); err != nil {
			return err
		}
	}

	return nil
}

func parsePathAdvanced(parts []string) (string, processingOptions, error) {
	var urlStart int

	po := defaultProcessingOptions()

	for i, part := range parts {
		args := strings.Split(part, ":")

		if len(args) == 1 {
			urlStart = i
			break
		}

		if err := applyProcessingOption(&po, args[0], args[1:]); err != nil {
			return "", po, err
		}
	}

	url, imgType, err := decodeUrl(parts[urlStart:])
	if err != nil {
		return "", po, err
	}

	if err := applyFormatOption(&po, imgType); err != nil {
		return "", po, errors.New("Resulting image type not supported")
	}

	return string(url), po, nil
}

func parsePathSimple(parts []string) (string, processingOptions, error) {
	var po processingOptions
	var err error

	if len(parts) < 6 {
		return "", po, errors.New("Invalid path")
	}

	po.Resize = resizeTypes[parts[0]]

	if err = applyWidthOption(&po, parts[1:2]); err != nil {
		return "", po, err
	}

	if err = applyHeightOption(&po, parts[2:3]); err != nil {
		return "", po, err
	}

	if err = applyGravityOption(&po, parts[3:4]); err != nil {
		return "", po, err
	}

	if err = applyEnlargeOption(&po, parts[4:5]); err != nil {
		return "", po, err
	}

	url, imgType, err := decodeUrl(parts[5:])
	if err != nil {
		return "", po, err
	}

	if err := applyFormatOption(&po, imgType); err != nil {
		return "", po, errors.New("Resulting image type not supported")
	}

	return string(url), po, nil
}

func parsePath(r *http.Request) (string, processingOptions, error) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	if len(parts) < 3 {
		return "", processingOptions{}, errors.New("Invalid path")
	}

	// if err := validatePath(parts[0], strings.TrimPrefix(path, fmt.Sprintf("/%s", parts[0]))); err != nil {
	// 	return "", processingOptions{}, err
	// }

	if _, ok := resizeTypes[parts[1]]; ok {
		return parsePathSimple(parts[1:])
	} else {
		return parsePathAdvanced(parts[1:])
	}
}
