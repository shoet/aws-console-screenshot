package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// ブラウザ起動
func BuildBrowser(browserPath string) (browser *rod.Browser, cleanup func() error, err error) {
	fmt.Println("get launcher")
	l := launcher.New().
		Bin(browserPath).
		// Headless(false).
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
		fmt.Println("terminate browser")
		if err := browser.Close(); err != nil {
			return fmt.Errorf("Failed to close browser: %w", err)
		}
		return nil
	}
	return browser, cleanup, nil
}

func LoginAWSConsole(browser *rod.Browser, accountId string, username string, password string) (*rod.Page, error) {
	// コンソールにアクセス
	url := fmt.Sprintf("https://%s.signin.aws.amazon.com/console", accountId)
	targetInput := proto.TargetCreateTarget{
		URL: url,
	}
	page, err := browser.Page(targetInput)
	if err != nil {
		return nil, fmt.Errorf("failed to browser page: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to WaitLoad: %v", err)
	}

	// ユーザー名を入力
	usernameElement, err := page.Element("input[type='text']#username")
	if err != nil {
		return nil, fmt.Errorf("failed to find username element: %v", err)
	}
	if err := usernameElement.Input(username); err != nil {
		return nil, fmt.Errorf("failed to input username: %v", err)
	}

	// パスワードを入力
	passwordElement, err := page.Element("input[type='password']#password")
	if err != nil {
		return nil, fmt.Errorf("failed to find password element: %v", err)
	}
	if err := passwordElement.Input(password); err != nil {
		return nil, fmt.Errorf("failed to input password: %v", err)
	}

	// ログインボタンをクリック
	loginButton, err := page.Element("a#signin_button")
	if err != nil {
		return nil, fmt.Errorf("failed to find login button: %v", err)
	}
	if err := loginButton.Tap(); err != nil {
		return nil, fmt.Errorf("failed to tap login button: %v", err)
	}

	return page, nil
}

func WaitPageStable(page *rod.Page) error {
	if err := page.WaitDOMStable(time.Second*3, 0.5); err != nil {
		return fmt.Errorf("failed to WaitLoad: %v", err)
	}
	return nil
}

func NavigatePage(page *rod.Page, url string) (*rod.Page, error) {
	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("failed to Navigate: %v", err)
	}
	return page, nil
}

func GetScreenShot(page *rod.Page) ([]byte, error) {
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed tod WaitLoad: %v", err)
	}
	// スクリーンショット
	data, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format: "png",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to page screenshot: %v", err)
	}
	return data, nil
}
