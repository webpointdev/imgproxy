# Serving files from S3

imgproxy can process images from S3 buckets. To use this feature, do the following:

1. Set `IMGPROXY_USE_S3` environment variable as `true`;
2. [Setup credentials](#setup-credentials) to grant access to your bucket;
3. _(optional)_ Specify AWS region with `IMGPROXY_S3_REGION` or `AWS_REGION`. Default: `us-west-1`;
4. _(optional)_ Specify S3 endpoint with `IMGPROXY_S3_ENDPOINT`;
5. Use `s3://%bucket_name/%file_key` as the source image URL.

If you need to specify version of the source object, you can use query string of the source URL:

```
s3://%bucket_name/%file_key?%version_id
```

### Setup credentials

There are three ways to specify your AWS credentials. The credentials need to have read rights for all of the buckets given in the source URLs.

#### Environment variables

You can specify AWS Acces Key ID and Secret Access Key by setting the standard `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables.

``` bash
AWS_ACCESS_KEY_ID=my_access_key AWS_SECRET_ACCESS_KEY=my_secret_key imgproxy

# same for Docker
docker run -e AWS_ACCESS_KEY_ID=my_access_key -e AWS_SECRET_ACCESS_KEY=my_secret_key -it darthsim/imgproxy
```

It is the recommended way to use with dockerized imgproxy.

#### Shared credentials file

Otherwise, you can create the `.aws/credentials` file in your home directory with the following content:

```ini
[default]
aws_access_key_id = %access_key_id
aws_secret_access_key = %secret_access_key
```

#### IAM Roles for Amazon EC2 Instances

If you are running imgproxy on an Amazon EC2 instance, you can use the instance's [IAM role](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html) to get security credentials to make calls to AWS S3.

You can learn about credentials in the [Configuring the AWS SDK for Go](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) guide.

## Minio

[Minio](https://github.com/minio/minio) is an object storage server released under Apache License v2.0. It is compatible with Amazon S3, so it can be used with imgproxy.

To use Minio as source images provider, do the following:

* Setup Amazon S3 support as usual using environment variables or shared config file;
* Specify endpoint with `IMGPROXY_S3_ENDPOINT`. Use `http://...` endpoint to disable SSL.
