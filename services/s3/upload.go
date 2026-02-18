package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Uploader 提供 S3 上傳能力
type Uploader struct {
	client *awss3.Client
	bucket string
}

// NewUploaderFromEnv 從環境變數建立 S3 Uploader
func NewUploaderFromEnv() (*Uploader, error) {
	bucket := os.Getenv("AWS_S3_BUCKET_NAME")
	region := os.Getenv("AWS_S3_REGION")
	accessKey := os.Getenv("AWS_S3_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_S3_SECRET_ACCESS_KEY")

	if bucket == "" || region == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET_NAME、AWS_S3_REGION、AWS_S3_ACCESS_KEY_ID、AWS_S3_SECRET_ACCESS_KEY 必須設定")
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	client := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})
	return &Uploader{client: client, bucket: bucket}, nil
}

// Upload 上傳圖片至 S3，回傳物件 Key 與錯誤
func (u *Uploader) Upload(ctx context.Context, userID string, body io.Reader, contentType string) (key string, err error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	if userID == "" {
		userID = "unknown"
	}
	timestamp := time.Now().Format("20060102_150405")
	key = fmt.Sprintf("food-images/%s/%s.jpg", userID, timestamp)

	_, err = u.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:        aws.String(u.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

// PresignGetURL 產生取得該物件的 Presigned URL（預設 1 小時有效）
func (u *Uploader) PresignGetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if expires <= 0 {
		expires = 1 * time.Hour
	}
	presignClient := awss3.NewPresignClient(u.client)
	presigned, err := presignClient.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	}, func(opts *awss3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", err
	}
	return presigned.URL, nil
}
