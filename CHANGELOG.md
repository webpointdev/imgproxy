# Changelog

## v2.1.0

- [Plain source URLs](./docs/generating_the_url_advanced.md#plain) support;
- [Serving images from Google Cloud Storage](./docs/serving_files_from_google_cloud_storage.md);
- [Full support of GIFs](./docs/image_formats_support.md) including animated ones;
- [Watermarks](./docs/watermark.md);
- [New Relic](./docs/new_relic.md) metrics;
- [Prometheus](./docs/prometheus.md) metrics;
- [Cache buster](./docs/generating_the_url_advanced.md#cache-buster) option;
- [Quality](./docs/generating_the_url_advanced.md#quality) option;
- Support for custom [Amazon S3](./docs/serving_files_from_s3.md) endpoints;
- Support for [Amazon S3](./docs/serving_files_from_s3.md) versioning;
- [Client hints](./docs/configuration.md#client-hints-support) support;
- Using source image format when one is not specified in the URL;
- Sending `User-Agent` header when downloading a source image;
- Setting proper filename in `Content-Disposition` header in the response;
- Truncated signature support.

## v2.0.3

Fixed URL validation when IMGPROXY_BASE_URL is used

## v2.0.2

Fixed smart crop + blur/sharpen SIGSEGV on Alpine

## v2.0.1

Minor fixes

## v2.0.0

All-You-Ever-Wanted release! :tada:

- Key and salt are not required anymore. When key or salt is not specified, signature checking is disabled;
- [New advanced URL format](./docs/generating_the_url_advanced.md). Unleash the full power of imgproxy v2.0;
- [Presets](./docs/presets.md). Shorten your urls by reusing processing options;
- [Serving images from Amazon S3](./docs/serving_files_from_s3.md). Thanks to [@crohr](https://github.com/crohr), now we have a way to serve files from private S3 buckets;
- [Autoconverting to WebP when supported by browser](./docs/configuration.md#webp-support-detection) (disabled by default). Use WebP as resulting format when browser supports it;
- [Gaussian blur](./docs/generating_the_url_advanced.md#blur) and [sharpen](./docs/generating_the_url_advanced.md#sharpen) filters. Make your images look better than before;
- [Focus point gravity](./docs/generating_the_url_advanced.md#gravity). Tell imgproxy what point will be the center of the image;
- [Background color](./docs/generating_the_url_advanced.md#background). Control the color of background when converting PNG with alpha-channel to JPEG;
- Imgproxy calcs resulting width/height automaticly when one specified as zero;
- Memory usage is optimized.

## v1.1.8

- Disabled libvips cache to prevent SIGSEGV on Alpine

## v1.1.7

- Improved ETag generation

## v1.1.6

- Added progressive JPEG and interlaced PNG support

## v1.1.5.1

- Fixed autorotation when image is not resized

## v1.1.5

- Add CORS headers
- Add IMGPROXY_BASE_URL config
- Add Content-Length header

## v1.1.4

- Added request ID
- Idle time does not causes timeout
- Increased default maximum number of simultaneous active connections
