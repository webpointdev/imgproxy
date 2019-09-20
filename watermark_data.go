package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
)

func getWatermarkData() (*imageData, error) {
	if len(conf.WatermarkData) > 0 {
		return base64WatermarkData(conf.WatermarkData)
	}

	if len(conf.WatermarkPath) > 0 {
		return fileWatermarkData(conf.WatermarkPath)
	}

	if len(conf.WatermarkURL) > 0 {
		return remoteWatermarkData(conf.WatermarkURL)
	}

	return nil, nil
}

func base64WatermarkData(encoded string) (*imageData, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("Can't decode watermark data: %s", err)
	}

	imgtype, err := checkTypeAndDimensions(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Can't decode watermark: %s", err)
	}

	return &imageData{Data: data, Type: imgtype}, nil
}

func fileWatermarkData(path string) (*imageData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Can't read watermark: %s", err)
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("Can't read watermark: %s", err)
	}

	imgdata, err := readAndCheckImage(f, int(fi.Size()))
	if err != nil {
		return nil, fmt.Errorf("Can't read watermark: %s", err)
	}

	return imgdata, err
}

func remoteWatermarkData(imageURL string) (*imageData, error) {
	res, err := requestImage(imageURL)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("Can't download watermark: %s", err)
	}

	imgdata, err := readAndCheckImage(res.Body, int(res.ContentLength))
	if err != nil {
		return nil, fmt.Errorf("Can't download watermark: %s", err)
	}

	return imgdata, err
}
