package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator/v10"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AwsAccountId string `envconfig:"AWS_ACCOUNT_ID" validate:"required"`
	AwsUsername  string `envconfig:"AWS_USERNAME" validate:"required"`
	AwsPassword  string `envconfig:"AWS_PASSWORD" validate:"required"`
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

	if err := LoginAWSConsole(browser, config.AwsAccountId, config.AwsUsername, config.AwsPassword); err != nil {
		return fmt.Errorf("failed to LoginAWSConsole: %v", err)
	}

	if err := cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup browser: %v", err)
	}
	return nil
}

// ブラウザ起動
func BuildBrowser(browserPath string) (browser *rod.Browser, cleanup func() error, err error) {
	fmt.Println("get launcher")
	l := launcher.New().
		Bin(browserPath).
		Headless(false).
		// Headless(true).
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

func LoginAWSConsole(browser *rod.Browser, accountId string, username string, password string) error {
	// コンソールにアクセス
	url := fmt.Sprintf("https://%s.signin.aws.amazon.com/console", accountId)
	targetInput := proto.TargetCreateTarget{
		URL: url,
	}
	page, err := browser.Page(targetInput)
	if err != nil {
		return fmt.Errorf("failed to browser page: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("failed tod WaitLoad: %v", err)
	}
	// ユーザー名を入力
	usernameElement, err := page.Element("input[type='text']#username")
	if err != nil {
		return fmt.Errorf("failed to find username element: %v", err)
	}
	if err := usernameElement.Input(username); err != nil {
		return fmt.Errorf("failed to input username: %v", err)
	}

	// パスワードを入力
	passwordElement, err := page.Element("input[type='password']#password")
	if err != nil {
		return fmt.Errorf("failed to find password element: %v", err)
	}
	if err := passwordElement.Input(password); err != nil {
		return fmt.Errorf("failed to input password: %v", err)
	}

	// ログインボタンをクリック
	loginButton, err := page.Element("a#signin_button")
	if err != nil {
		return fmt.Errorf("failed to find login button: %v", err)
	}
	if err := loginButton.Tap(); err != nil {
		return fmt.Errorf("failed to tap login button: %v", err)
	}

	return nil
}

// URLにアクセス
// スクリーンショット
// S3に保存

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
