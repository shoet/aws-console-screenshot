package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AwsAccountId       string `envconfig:"ACCOUNTID" validate:"required"`
	AwsUsername        string `envconfig:"USERNAME" validate:"required"`
	AwsPassword        string `envconfig:"PASSWORD" validate:"required"`
	AwsRegion          string `envconfig:"REGION" validate:"required"`
	AwsS3Bucket        string `envconfig:"BUCKET_NAME" validate:"required"`
	AwsS3ImageSavePath string `envconfig:"IMAGE_SAVE_PATH" validate:"required"`
	BrowserPath        string `envconfig:"BROWSER_PATH" default:"/opt/homebrew/bin/chromium"`
	LocalStoragePath   string `envconfig:"LOCAL_STORAGE_PATH" default:"data"`
}

func LoadConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("failed to LoadConfig: %v", err)
	}
	v := validator.New()
	if err := v.Struct(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}
	return &config, nil
}

type ScreenCapture struct {
	config       *Config
	localStorage *LocalStorage
	s3Adapter    *S3Adapter
}

func NewScreenCapture(config *Config, localStorage *LocalStorage, s3Adapter *S3Adapter) *ScreenCapture {
	return &ScreenCapture{
		config:       config,
		localStorage: localStorage,
		s3Adapter:    s3Adapter,
	}
}

func (s *ScreenCapture) Run(context context.Context, config *Config) error {
	// ブラウザ起動
	browser, cleanup, err := BuildBrowser(config.BrowserPath)
	if err != nil {
		return fmt.Errorf("failed to BuildBrowser: %v", err)
	}

	// ログイン
	page, err := LoginAWSConsole(browser, config.AwsAccountId, config.AwsUsername, config.AwsPassword)
	if err != nil {
		return fmt.Errorf("failed to LoginAWSConsole: %v", err)
	}

	// ロード待機
	if err := WaitPageStable(page); err != nil {
		return fmt.Errorf("failed to WaitPageStable: %v", err)
	}

	// コンソールにアクセス
	url := fmt.Sprintf("https://%[1]s.console.aws.amazon.com/console/home?region=%[1]s", config.AwsRegion)
	page, err = NavigatePage(page, url)
	if err != nil {
		return fmt.Errorf("failed to LoadConsolePage: %v", err)
	}

	// ロード待機
	if err := WaitPageStable(page); err != nil {
		return fmt.Errorf("failed to WaitPageStable: %v", err)
	}

	urls := []string{
		url, // TODO: test
	}

	for _, url := range urls {
		// 対象のページに遷移
		page, err = NavigatePage(page, url)
		if err != nil {
			return fmt.Errorf("failed to NavigatePage: %v", err)
		}

		// ロード待機
		if err := WaitPageStable(page); err != nil {
			return fmt.Errorf("failed to WaitPageStable: %v", err)
		}

		// スクリーンショット
		pngBinary, err := GetScreenShot(page)
		if err != nil {
			return fmt.Errorf("failed to GetScreenShot URL[%s]: %w", url, err)
		}

		// 画像保存
		n := time.Now().Add(time.Hour * 9)
		fileName := strings.ReplaceAll(n.Format("20060102150405.000"), ".", "") + ".png"
		if err := s.SaveImage(pngBinary, fileName); err != nil {
			return fmt.Errorf("failed to SaveImage: %v", err)
		}
	}

	if err := cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup browser: %v", err)
	}
	return nil
}

func (s *ScreenCapture) SaveImage(data []byte, fileName string) error {
	r := bytes.NewReader(data)
	if len(os.Args) > 1 && os.Args[1] == "local" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("faild to Getwd: %v", err)
		}

		outputDir := filepath.Join(cwd, s.config.LocalStoragePath)
		if _, err := os.Stat(outputDir); err != nil {
			if !os.IsExist(err) {
				return fmt.Errorf("failed to os.Stat: %v", err)
			}
			if err := os.Mkdir(outputDir, 0755); err != nil {
				return fmt.Errorf("faild tod Mkdir: %v", err)
			}
		}

		filePath := filepath.Join(outputDir, fileName)
		if err := s.localStorage.SaveFile(bytes.NewReader(data), filePath); err != nil {
			return fmt.Errorf("failed to SaveFile: %v", err)
		}
	} else {
		key := filepath.Join(s.config.AwsS3ImageSavePath, fileName)
		if err := s.s3Adapter.SaveFile(r, s.config.AwsS3Bucket, key); err != nil {
			return fmt.Errorf("failed to SaveFile: %v", err)
		}
	}
	return nil
}

func main() {
	if _, err := os.Stat(".env"); err != nil {
		fmt.Println("not found dotenv")
	} else {
		if err := godotenv.Load(".env"); err != nil {
			panic(err)
		}
	}
	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	localStorage, err := NewLocalStorage()
	if err != nil {
		panic(err)
	}

	cfg, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}
	s3Adapter, err := NewS3Adapter(&S3AdapterInput{
		AwsConfig: &cfg,
	})
	if err != nil {
		panic(err)
	}

	cap := NewScreenCapture(config, localStorage, s3Adapter)

	if len(os.Args) > 1 && os.Args[1] == "local" {
		if err = cap.Run(context.Background(), config); err != nil {
			panic(err)
		}
	} else {
		lambda.Start(func(ctx context.Context) error {
			return cap.Run(ctx, config)
		})
	}
}
