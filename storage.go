package main

import (
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Adapter struct {
	s3Client *s3.Client
}

func NewS3Adapter() (*S3Adapter, error) {
	return &S3Adapter{}, nil
}

func (s *S3Adapter) SaveFile(reader io.Reader, bucket string, key string) error {
	fmt.Println("saved s3")
	return nil
}

type LocalStorage struct{}

func NewLocalStorage() (*LocalStorage, error) {
	return &LocalStorage{}, nil
}

func (s *LocalStorage) SaveFile(reader io.Reader, filePath string) error {
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
