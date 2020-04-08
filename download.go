package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/imgproxy/imgproxy/v2/imagemeta"
)

var (
	downloadClient *http.Client

	imageDataCtxKey          = ctxKey("imageData")
	cacheControlHeaderCtxKey = ctxKey("cacheControlHeader")
	expiresHeaderCtxKey      = ctxKey("expiresHeader")

	errSourceDimensionsTooBig      = newError(422, "Source image dimensions are too big", "Invalid source image")
	errSourceResolutionTooBig      = newError(422, "Source image resolution is too big", "Invalid source image")
	errSourceFileTooBig            = newError(422, "Source image file is too big", "Invalid source image")
	errSourceImageTypeNotSupported = newError(422, "Source image type not supported", "Invalid source image")
)

const msgSourceImageIsUnreachable = "Source image is unreachable"

var downloadBufPool *bufPool

type limitReader struct {
	r    io.Reader
	left int
}

func (lr *limitReader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	lr.left -= n

	if err == nil && lr.left < 0 {
		err = errSourceFileTooBig
	}

	return
}

func initDownloading() error {
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        conf.Concurrency,
		MaxIdleConnsPerHost: conf.Concurrency,
		DisableCompression:  true,
		Dial:                (&net.Dialer{KeepAlive: 600 * time.Second}).Dial,
	}

	if conf.IgnoreSslVerification {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if conf.LocalFileSystemRoot != "" {
		transport.RegisterProtocol("local", newFsTransport())
	}

	if conf.S3Enabled {
		if t, err := newS3Transport(); err != nil {
			return err
		} else {
			transport.RegisterProtocol("s3", t)
		}
	}

	if conf.GCSEnabled {
		if t, err := newGCSTransport(); err != nil {
			return err
		} else {
			transport.RegisterProtocol("gs", t)
		}
	}

	downloadClient = &http.Client{
		Timeout:   time.Duration(conf.DownloadTimeout) * time.Second,
		Transport: transport,
	}

	downloadBufPool = newBufPool("download", conf.Concurrency, conf.DownloadBufferSize)

	imagemeta.SetMaxSvgCheckRead(conf.MaxSvgCheckBytes)

	return nil
}

func checkDimensions(width, height int) error {
	if conf.MaxSrcDimension > 0 && (width > conf.MaxSrcDimension || height > conf.MaxSrcDimension) {
		return errSourceDimensionsTooBig
	}

	if width*height > conf.MaxSrcResolution {
		return errSourceResolutionTooBig
	}

	return nil
}

func checkTypeAndDimensions(r io.Reader) (imageType, error) {
	meta, err := imagemeta.DecodeMeta(r)
	if err == imagemeta.ErrFormat {
		return imageTypeUnknown, errSourceImageTypeNotSupported
	}
	if err != nil {
		return imageTypeUnknown, newUnexpectedError(err.Error(), 0)
	}

	imgtype, imgtypeOk := imageTypes[meta.Format()]
	if !imgtypeOk || !imageTypeLoadSupport(imgtype) {
		return imageTypeUnknown, errSourceImageTypeNotSupported
	}

	if err = checkDimensions(meta.Width(), meta.Height()); err != nil {
		return imageTypeUnknown, err
	}

	return imgtype, nil
}

func readAndCheckImage(r io.Reader, contentLength int) (*imageData, error) {
	if conf.MaxSrcFileSize > 0 && contentLength > conf.MaxSrcFileSize {
		return nil, errSourceFileTooBig
	}

	buf := downloadBufPool.Get(contentLength)
	cancel := func() { downloadBufPool.Put(buf) }

	if conf.MaxSrcFileSize > 0 {
		r = &limitReader{r: r, left: conf.MaxSrcFileSize}
	}

	imgtype, err := checkTypeAndDimensions(io.TeeReader(r, buf))
	if err != nil {
		cancel()
		return nil, err
	}

	if _, err = buf.ReadFrom(r); err != nil {
		cancel()
		return nil, newError(404, err.Error(), msgSourceImageIsUnreachable)
	}

	return &imageData{buf.Bytes(), imgtype, cancel}, nil
}

func requestImage(imageURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, newError(404, err.Error(), msgSourceImageIsUnreachable).SetUnexpected(conf.ReportDownloadingErrors)
	}

	req.Header.Set("User-Agent", conf.UserAgent)

	res, err := downloadClient.Do(req)
	if err != nil {
		return res, newError(404, err.Error(), msgSourceImageIsUnreachable).SetUnexpected(conf.ReportDownloadingErrors)
	}

	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		msg := fmt.Sprintf("Can't download image; Status: %d; %s", res.StatusCode, string(body))
		return res, newError(404, msg, msgSourceImageIsUnreachable).SetUnexpected(conf.ReportDownloadingErrors)
	}

	return res, nil
}

func downloadImage(ctx context.Context) (context.Context, context.CancelFunc, error) {
	imageURL := getImageURL(ctx)

	if newRelicEnabled {
		newRelicCancel := startNewRelicSegment(ctx, "Downloading image")
		defer newRelicCancel()
	}

	if prometheusEnabled {
		defer startPrometheusDuration(prometheusDownloadDuration)()
	}

	res, err := requestImage(imageURL)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return ctx, func() {}, err
	}

	imgdata, err := readAndCheckImage(res.Body, int(res.ContentLength))
	if err != nil {
		return ctx, func() {}, err
	}

	ctx = context.WithValue(ctx, imageDataCtxKey, imgdata)
	ctx = context.WithValue(ctx, cacheControlHeaderCtxKey, res.Header.Get("Cache-Control"))
	ctx = context.WithValue(ctx, expiresHeaderCtxKey, res.Header.Get("Expires"))

	return ctx, imgdata.Close, err
}

func getImageData(ctx context.Context) *imageData {
	return ctx.Value(imageDataCtxKey).(*imageData)
}

func getCacheControlHeader(ctx context.Context) string {
	str, _ := ctx.Value(cacheControlHeaderCtxKey).(string)
	return str
}

func getExpiresHeader(ctx context.Context) string {
	str, _ := ctx.Value(expiresHeaderCtxKey).(string)
	return str
}
