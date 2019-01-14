package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func intEnvConfig(i *int, name string) {
	if env, err := strconv.Atoi(os.Getenv(name)); err == nil {
		*i = env
	}
}

func floatEnvConfig(i *float64, name string) {
	if env, err := strconv.ParseFloat(os.Getenv(name), 64); err == nil {
		*i = env
	}
}

func megaIntEnvConfig(f *int, name string) {
	if env, err := strconv.ParseFloat(os.Getenv(name), 64); err == nil {
		*f = int(env * 1000000)
	}
}

func strEnvConfig(s *string, name string) {
	if env := os.Getenv(name); len(env) > 0 {
		*s = env
	}
}

func boolEnvConfig(b *bool, name string) {
	*b = false
	if env, err := strconv.ParseBool(os.Getenv(name)); err == nil {
		*b = env
	}
}

func hexEnvConfig(b *[]securityKey, name string) {
	var err error

	if env := os.Getenv(name); len(env) > 0 {
		parts := strings.Split(env, ",")

		keys := make([]securityKey, len(parts))

		for i, part := range parts {
			if keys[i], err = hex.DecodeString(part); err != nil {
				log.Fatalf("%s expected to be hex-encoded strings. Invalid: %s\n", name, part)
			}
		}

		*b = keys
	}
}

func hexFileConfig(b *[]securityKey, filepath string) {
	if len(filepath) == 0 {
		return
	}

	f, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("Can't open file %s\n", filepath)
	}

	keys := []securityKey{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		part := scanner.Text()

		if len(part) == 0 {
			continue
		}

		if key, err := hex.DecodeString(part); err == nil {
			keys = append(keys, key)
		} else {
			log.Fatalf("%s expected to contain hex-encoded strings. Invalid: %s\n", filepath, part)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read file %s: %s", filepath, err)
	}

	*b = keys
}

func presetEnvConfig(p presets, name string) {
	if env := os.Getenv(name); len(env) > 0 {
		presetStrings := strings.Split(env, ",")

		for _, presetStr := range presetStrings {
			if err := parsePreset(p, presetStr); err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func presetFileConfig(p presets, filepath string) {
	if len(filepath) == 0 {
		return
	}

	f, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("Can't open file %s\n", filepath)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := parsePreset(p, scanner.Text()); err != nil {
			log.Fatalln(err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read presets file: %s", err)
	}
}

type config struct {
	Bind            string
	ReadTimeout     int
	WaitTimeout     int
	WriteTimeout    int
	DownloadTimeout int
	Concurrency     int
	MaxClients      int
	TTL             int

	MaxSrcDimension  int
	MaxSrcResolution int
	MaxGifFrames     int

	JpegProgressive bool
	PngInterlaced   bool
	Quality         int
	GZipCompression int

	EnableWebpDetection bool
	EnforceWebp         bool
	EnableClientHints   bool

	Keys          []securityKey
	Salts         []securityKey
	AllowInsecure bool
	SignatureSize int

	Secret string

	AllowOrigin string

	UserAgent string

	IgnoreSslVerification bool

	LocalFileSystemRoot string
	S3Enabled           bool
	S3Region            string
	S3Endpoint          string
	GCSKey              string

	ETagEnabled bool

	BaseURL string

	Presets presets

	WatermarkData    string
	WatermarkPath    string
	WatermarkURL     string
	WatermarkOpacity float64

	NewRelicAppName string
	NewRelicKey     string

	PrometheusBind string

	BugsnagKey        string
	BugsnagStage      string
	HoneybadgerKey    string
	HoneybadgerEnv    string
	SentryDSN         string
	SentryEnvironment string
	SentryRelease     string
}

var conf = config{
	Bind:                  ":8080",
	ReadTimeout:           10,
	WriteTimeout:          10,
	DownloadTimeout:       5,
	Concurrency:           runtime.NumCPU() * 2,
	TTL:                   3600,
	IgnoreSslVerification: false,
	MaxSrcResolution:      16800000,
	MaxGifFrames:          1,
	AllowInsecure:         false,
	SignatureSize:         32,
	Quality:               80,
	GZipCompression:       5,
	UserAgent:             fmt.Sprintf("imgproxy/%s", version),
	ETagEnabled:           false,
	S3Enabled:             false,
	WatermarkOpacity:      1,
	BugsnagStage:          "production",
	HoneybadgerEnv:        "production",
	SentryEnvironment:     "production",
	SentryRelease:         fmt.Sprintf("imgproxy/%s", version),
}

func init() {
	keyPath := flag.String("keypath", "", "path of the file with hex-encoded key")
	saltPath := flag.String("saltpath", "", "path of the file with hex-encoded salt")
	presetsPath := flag.String("presets", "", "path of the file with presets")
	showVersion := flag.Bool("v", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if port := os.Getenv("PORT"); len(port) > 0 {
		conf.Bind = fmt.Sprintf(":%s", port)
	}

	strEnvConfig(&conf.Bind, "IMGPROXY_BIND")
	intEnvConfig(&conf.ReadTimeout, "IMGPROXY_READ_TIMEOUT")
	intEnvConfig(&conf.WriteTimeout, "IMGPROXY_WRITE_TIMEOUT")
	intEnvConfig(&conf.DownloadTimeout, "IMGPROXY_DOWNLOAD_TIMEOUT")
	intEnvConfig(&conf.Concurrency, "IMGPROXY_CONCURRENCY")
	intEnvConfig(&conf.MaxClients, "IMGPROXY_MAX_CLIENTS")

	intEnvConfig(&conf.TTL, "IMGPROXY_TTL")

	intEnvConfig(&conf.MaxSrcDimension, "IMGPROXY_MAX_SRC_DIMENSION")
	megaIntEnvConfig(&conf.MaxSrcResolution, "IMGPROXY_MAX_SRC_RESOLUTION")
	intEnvConfig(&conf.MaxGifFrames, "IMGPROXY_MAX_GIF_FRAMES")

	boolEnvConfig(&conf.JpegProgressive, "IMGPROXY_JPEG_PROGRESSIVE")
	boolEnvConfig(&conf.PngInterlaced, "IMGPROXY_PNG_INTERLACED")
	intEnvConfig(&conf.Quality, "IMGPROXY_QUALITY")
	intEnvConfig(&conf.GZipCompression, "IMGPROXY_GZIP_COMPRESSION")

	boolEnvConfig(&conf.EnableWebpDetection, "IMGPROXY_ENABLE_WEBP_DETECTION")
	boolEnvConfig(&conf.EnforceWebp, "IMGPROXY_ENFORCE_WEBP")
	boolEnvConfig(&conf.EnableClientHints, "IMGPROXY_ENABLE_CLIENT_HINTS")

	hexEnvConfig(&conf.Keys, "IMGPROXY_KEY")
	hexEnvConfig(&conf.Salts, "IMGPROXY_SALT")
	intEnvConfig(&conf.SignatureSize, "IMGPROXY_SIGNATURE_SIZE")

	hexFileConfig(&conf.Keys, *keyPath)
	hexFileConfig(&conf.Salts, *saltPath)

	strEnvConfig(&conf.Secret, "IMGPROXY_SECRET")

	strEnvConfig(&conf.AllowOrigin, "IMGPROXY_ALLOW_ORIGIN")

	strEnvConfig(&conf.UserAgent, "IMGPROXY_USER_AGENT")

	boolEnvConfig(&conf.IgnoreSslVerification, "IMGPROXY_IGNORE_SSL_VERIFICATION")

	strEnvConfig(&conf.LocalFileSystemRoot, "IMGPROXY_LOCAL_FILESYSTEM_ROOT")

	boolEnvConfig(&conf.S3Enabled, "IMGPROXY_USE_S3")
	strEnvConfig(&conf.S3Region, "IMGPROXY_S3_REGION")
	strEnvConfig(&conf.S3Endpoint, "IMGPROXY_S3_ENDPOINT")

	strEnvConfig(&conf.GCSKey, "IMGPROXY_GCS_KEY")

	boolEnvConfig(&conf.ETagEnabled, "IMGPROXY_USE_ETAG")

	strEnvConfig(&conf.BaseURL, "IMGPROXY_BASE_URL")

	conf.Presets = make(presets)
	presetEnvConfig(conf.Presets, "IMGPROXY_PRESETS")
	presetFileConfig(conf.Presets, *presetsPath)

	strEnvConfig(&conf.WatermarkData, "IMGPROXY_WATERMARK_DATA")
	strEnvConfig(&conf.WatermarkPath, "IMGPROXY_WATERMARK_PATH")
	strEnvConfig(&conf.WatermarkURL, "IMGPROXY_WATERMARK_URL")
	floatEnvConfig(&conf.WatermarkOpacity, "IMGPROXY_WATERMARK_OPACITY")

	strEnvConfig(&conf.NewRelicAppName, "IMGPROXY_NEW_RELIC_APP_NAME")
	strEnvConfig(&conf.NewRelicKey, "IMGPROXY_NEW_RELIC_KEY")

	strEnvConfig(&conf.PrometheusBind, "IMGPROXY_PROMETHEUS_BIND")

	strEnvConfig(&conf.BugsnagKey, "IMGPROXY_BUGSNAG_KEY")
	strEnvConfig(&conf.BugsnagStage, "IMGPROXY_BUGSNAG_STAGE")
	strEnvConfig(&conf.HoneybadgerKey, "IMGPROXY_HONEYBADGER_KEY")
	strEnvConfig(&conf.HoneybadgerEnv, "IMGPROXY_HONEYBADGER_ENV")
	strEnvConfig(&conf.SentryDSN, "IMGPROXY_SENTRY_DSN")
	strEnvConfig(&conf.SentryEnvironment, "IMGPROXY_SENTRY_ENVIRONMENT")
	strEnvConfig(&conf.SentryRelease, "IMGPROXY_SENTRY_RELEASE")

	if len(conf.Keys) != len(conf.Salts) {
		log.Fatalf("Number of keys and number of salts should be equal. Keys: %d, salts: %d", len(conf.Keys), len(conf.Salts))
	}
	if len(conf.Keys) == 0 {
		warning("No keys defined, so signature checking is disabled")
		conf.AllowInsecure = true
	}
	if len(conf.Salts) == 0 {
		warning("No salts defined, so signature checking is disabled")
		conf.AllowInsecure = true
	}

	if conf.SignatureSize < 1 || conf.SignatureSize > 32 {
		log.Fatalf("Signature size should be within 1 and 32, now - %d\n", conf.SignatureSize)
	}

	if len(conf.Bind) == 0 {
		log.Fatalln("Bind address is not defined")
	}

	if conf.ReadTimeout <= 0 {
		log.Fatalf("Read timeout should be greater than 0, now - %d\n", conf.ReadTimeout)
	}

	if conf.WriteTimeout <= 0 {
		log.Fatalf("Write timeout should be greater than 0, now - %d\n", conf.WriteTimeout)
	}

	if conf.DownloadTimeout <= 0 {
		log.Fatalf("Download timeout should be greater than 0, now - %d\n", conf.DownloadTimeout)
	}

	if conf.Concurrency <= 0 {
		log.Fatalf("Concurrency should be greater than 0, now - %d\n", conf.Concurrency)
	}

	if conf.MaxClients <= 0 {
		conf.MaxClients = conf.Concurrency * 10
	}

	if conf.TTL <= 0 {
		log.Fatalf("TTL should be greater than 0, now - %d\n", conf.TTL)
	}

	if conf.MaxSrcDimension < 0 {
		log.Fatalf("Max src dimension should be greater than or equal to 0, now - %d\n", conf.MaxSrcDimension)
	} else if conf.MaxSrcDimension > 0 {
		warning("IMGPROXY_MAX_SRC_DIMENSION is deprecated and can be removed in future versions. Use IMGPROXY_MAX_SRC_RESOLUTION")
	}

	if conf.MaxSrcResolution <= 0 {
		log.Fatalf("Max src resolution should be greater than 0, now - %d\n", conf.MaxSrcResolution)
	}

	if conf.MaxGifFrames <= 0 {
		log.Fatalf("Max GIF frames should be greater than 0, now - %d\n", conf.MaxGifFrames)
	}

	if conf.Quality <= 0 {
		log.Fatalf("Quality should be greater than 0, now - %d\n", conf.Quality)
	} else if conf.Quality > 100 {
		log.Fatalf("Quality can't be greater than 100, now - %d\n", conf.Quality)
	}

	if conf.GZipCompression < 0 {
		log.Fatalf("GZip compression should be greater than or quual to 0, now - %d\n", conf.GZipCompression)
	} else if conf.GZipCompression > 9 {
		log.Fatalf("GZip compression can't be greater than 9, now - %d\n", conf.GZipCompression)
	}

	if conf.IgnoreSslVerification {
		warning("Ignoring SSL verification is very unsafe")
	}

	if conf.LocalFileSystemRoot != "" {
		stat, err := os.Stat(conf.LocalFileSystemRoot)
		if err != nil {
			log.Fatalf("Cannot use local directory: %s", err)
		} else {
			if !stat.IsDir() {
				log.Fatalf("Cannot use local directory: not a directory")
			}
		}
		if conf.LocalFileSystemRoot == "/" {
			log.Print("Exposing root via IMGPROXY_LOCAL_FILESYSTEM_ROOT is unsafe")
		}
	}

	if err := checkPresets(conf.Presets); err != nil {
		log.Fatalln(err)
	}

	if conf.WatermarkOpacity <= 0 {
		log.Fatalln("Watermark opacity should be greater than 0")
	} else if conf.WatermarkOpacity > 1 {
		log.Fatalln("Watermark opacity should be less than or equal to 1")
	}

	if len(conf.PrometheusBind) > 0 && conf.PrometheusBind == conf.Bind {
		log.Fatalln("Can't use the same binding for the main server and Prometheus")
	}

	initDownloading()
	initNewrelic()
	initPrometheus()
	initErrorsReporting()
	initVips()
}
