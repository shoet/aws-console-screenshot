package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AwsAccountId string `envconfig:"AWS_ACCOUNT_ID" validate:"required"`
	AwsUsername  string `envconfig:"AWS_USERNAME" validate:"required"`
	AwsPassword  string `envconfig:"AWS_PASSWORD" validate:"required"`
	AwsRegion    string `envconfig:"AWS_REGION" validate:"required"`
	BrowserPath  string `envconfig:"BROWSER_PATH" default:"/opt/homebrew/bin/chromium"`
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

	page, err := LoginAWSConsole(browser, config.AwsAccountId, config.AwsUsername, config.AwsPassword)
	if err != nil {
		return fmt.Errorf("failed to LoginAWSConsole: %v", err)
	}

	if err := WaitPageStable(page); err != nil {
		return fmt.Errorf("failed to WaitPageStable: %v", err)
	}

	// コンソールにアクセス
	url := fmt.Sprintf("https://%[1]s.console.aws.amazon.com/console/home?region=%[1]s", config.AwsRegion)
	page, err = NavigatePage(page, url)
	if err != nil {
		return fmt.Errorf("failed to LoadConsolePage: %v", err)
	}

	if err := WaitPageStable(page); err != nil {
		return fmt.Errorf("failed to WaitPageStable: %v", err)
	}

	img, err := GetScreenShot(page)
	if err != nil {
		return fmt.Errorf("failed to GetScreenShot")
	}

	SaveImage(img, "./data.png")

	_ = page

	urls := []string{}

	for _, url := range urls {
		page, err = NavigatePage(page, url)
		if err != nil {
			return fmt.Errorf("failed to NavigatePage: %v", err)
		}

		if err := WaitPageStable(page); err != nil {
			return fmt.Errorf("failed to WaitPageStable: %v", err)
		}

		pngBinary, err := GetScreenShot(page)
		if err != nil {
			return fmt.Errorf("failed to GetScreenShot URL[%s]: %w", url, err)
		}
		// S3に保存
		fmt.Println(pngBinary[:3])
		if err := SaveImage(pngBinary, "./data.png"); err != nil {
			return fmt.Errorf("failed to SaveImage: %v", err)
		}
	}

	if err := cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup browser: %v", err)
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
