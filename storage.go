package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Adapter struct {
	s3Client *s3.Client
}

type S3AdapterInput struct {
	AwsConfig *aws.Config
}

func NewS3Adapter(input *S3AdapterInput) (*S3Adapter, error) {
	s3Client := s3.NewFromConfig(*input.AwsConfig, func(o *s3.Options) {
		o.RetryMaxAttempts = 3
	})
	return &S3Adapter{
		s3Client: s3Client,
	}, nil
}

func (s *S3Adapter) SaveFile(reader io.ReadSeeker, bucket string, key string) error {
	length, err := GetReaderLength(reader)
	if err != nil {
		return fmt.Errorf("failed GetReaderLength: %v", err)
	}
	destinationUrl, err := s.UploadFile(context.Background(), bucket, key, reader, length, "image/png")
	if err != nil {
		return fmt.Errorf("faield UploadFile: %v", err)
	}
	fmt.Printf("saved s3: %s\n", destinationUrl)
	return nil
}

func GetReaderLength(r io.ReadSeeker) (int64, error) {
	n, err := io.Copy(io.Discard, r)
	if err != nil {
		return 0, fmt.Errorf("faild Copy: %v", err)
	}
	if _, err := r.Seek(0, 0); err != nil {
		return 0, fmt.Errorf("faild Seek: %v", err)
	}
	return n, nil
}

func (s *S3Adapter) UploadFile(
	ctx context.Context,
	bucketName string,
	key string,
	body io.Reader,
	length int64,
	contentType string,
) (savedUrl string, err error) {
	input := &s3.PutObjectInput{
		Bucket:        &bucketName,
		Key:           &key,
		Body:          body,
		ContentLength: &length,
		ContentType:   aws.String(contentType),
	}
	if _, err := s.s3Client.PutObject(ctx, input,
		s3.WithAPIOptions(
			v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware,
		),
	); err != nil {
		return "", fmt.Errorf("failed to put object: %w", err)
	}

	destinationPath := fmt.Sprintf("%s/%s", bucketName, key)
	return destinationPath, nil
}

type LocalStorage struct{}

func NewLocalStorage() (*LocalStorage, error) {
	return &LocalStorage{}, nil
}

func (s *LocalStorage) SaveFile(reader io.ReadSeeker, filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("falied to Copy: %v", err)
	}
	return nil
}
