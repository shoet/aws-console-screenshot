package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator/v10"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AwsUsername string `validate:"required"`
	AwsPassword string `validate:"required"`
	BrowserPath string `envDefault:"/opt/homebrew/bin/chromium"`
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

// lambda関数
func run(context context.Context, config *Config) error {
	browser, cleanup, err := BuildBrowser(config.BrowserPath)
	if err != nil {
		return fmt.Errorf("failed to BuildBrowser: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			panic(err)
		}
	}()
	_ = browser
	return nil
}

func main() {
	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "local" {
		if err = run(context.Background(), config); err != nil {
			panic(err)
		}
	} else {
		lambda.Start(func(ctx context.Context) error {
			return run(ctx, config)
		})
	}
}

// ブラウザ起動
func BuildBrowser(browserPath string) (browser *rod.Browser, cleanup func() error, err error) {
	fmt.Println("get launcher")
	l := launcher.New().
		Bin(browserPath).
		Headless(true).
		NoSandbox(true).
		Set("disable-gpu", "").
		Set("disable-software-rasterizer", "").
		Set("single-process", "").
		Set("homedir", "/tmp").
		Set("data-path", "/tmp/data-path").
		Set("disk-cache-dir", "/tmp/cache-dir")

	launchArgs := l.FormatArgs()
	fmt.Printf("launchArgs: %s\n", launchArgs)

	fmt.Println("start launcher")
	url, err := l.Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to launch browser: %w", err)
	}

	fmt.Printf("url: %s\n", url)
	browser = rod.New().ControlURL(url)
	// .Trace(true)
	fmt.Println("start connect rod")
	if err := browser.Connect(); err != nil {
		return nil, nil, fmt.Errorf("Failed to connect to browser: %w", err)
	}
	fmt.Println("connected rod")

	cleanup = func() error {
		if err := browser.Close(); err != nil {
			return fmt.Errorf("Failed to close browser: %w", err)
		}
		return nil
	}

	return browser, cleanup, nil
}

// envからID/PASS取得
// コンソールにアクセス
// コンソールにログイン
// ログのURLにアクセス
// スクリーンショット
// S3に保存
