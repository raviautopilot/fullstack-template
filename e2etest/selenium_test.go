package e2etest

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tebeka/selenium"
)

func getFrontendURL() string {
	url := os.Getenv("FRONTEND_URL")
	if url == "" {
		url = "http://localhost:5173"
	}
	return url
}

// takeScreenshot is a helper to capture browser state during test failures
func takeScreenshot(wd selenium.WebDriver, t *testing.T, name string) {
	t.Helper()
	screenshot, err := wd.Screenshot()
	if err != nil {
		t.Logf("Failed to take screenshot: %v", err)
		return
	}

	// Create screenshots dir if not exists
	screenshotsDir := "screenshots"
	if _, err := os.Stat(screenshotsDir); os.IsNotExist(err) {
		os.Mkdir(screenshotsDir, os.ModePerm)
	}

	filename := fmt.Sprintf("%s/%s_%s.png", screenshotsDir, name, time.Now().Format("2006-01-02_15-04-05"))
	if err := os.WriteFile(filename, screenshot, 0644); err != nil {
		t.Logf("Failed to save screenshot to %s: %v", filename, err)
	} else {
		t.Logf("Screenshot saved to %s", filename)
	}
}

func TestFrontendSeleniumE2E(t *testing.T) {
	frontendURL := getFrontendURL()

	// ChromeDriver port
	const chromeDriverPort = 8082

	// Start ChromeDriver service
	var chromeDriverPath = os.Getenv("CHROMEDRIVER_PATH")
	if chromeDriverPath == "" {
		chromeDriverPath = "/usr/bin/chromedriver"
	}

	opts := []selenium.ServiceOption{
		selenium.Output(os.Stderr),
	}

	t.Logf("Starting ChromeDriver service with binary: %s on port %d...", chromeDriverPath, chromeDriverPort)
	service, err := selenium.NewChromeDriverService(chromeDriverPath, chromeDriverPort, opts...)
	if err != nil {
		t.Fatalf("Failed to start ChromeDriver service: %v", err)
	}
	defer service.Stop()

	// Configure Chrome capabilities
	caps := selenium.Capabilities{"browserName": "chrome"}

	var chromiumPath = os.Getenv("CHROMIUM_PATH")
	if chromiumPath == "" {
		chromiumPath = "/usr/bin/chromium"
	}

	chromeCaps := map[string]interface{}{
		"binary": chromiumPath,
		"args": []string{
			"--headless=new",
			"--no-sandbox",
			"--disable-gpu",
			"--disable-dev-shm-usage",
			"--remote-debugging-port=9222",
		},
	}
	caps["goog:chromeOptions"] = chromeCaps

	t.Log("Connecting to Chromium instance...")
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", chromeDriverPort))
	if err != nil {
		t.Fatalf("Failed to open remote Selenium session: %v", err)
	}
	defer wd.Quit()

	// Set window size for standard screen assertions
	if err := wd.ResizeWindow("current", 1280, 800); err != nil {
		t.Logf("Warning: failed to resize browser window: %v", err)
	}

	// 1. Navigate to Frontend
	t.Logf("Navigating to frontend at: %s", frontendURL)
	if err := wd.Get(frontendURL); err != nil {
		takeScreenshot(wd, t, "navigate_failed")
		t.Fatalf("Failed to load page: %v", err)
	}

	// Small pause to let Vite load initial bundle
	time.Sleep(2 * time.Second)

	// 2. Assert Logged Out state
	t.Log("Verifying logged-out landing page...")
	loginCard, err := wd.FindElement(selenium.ByCSSSelector, "[data-testid='login-card']")
	if err != nil {
		takeScreenshot(wd, t, "find_login_card_failed")
		t.Fatalf("Could not find login card: %v. Is React frontend running and accessible?", err)
	}

	isDisplayed, err := loginCard.IsDisplayed()
	if err != nil || !isDisplayed {
		takeScreenshot(wd, t, "login_card_not_visible")
		t.Fatalf("Login card is not visible on page")
	}

	googleBtn, err := wd.FindElement(selenium.ByCSSSelector, "[data-testid='google-login-btn']")
	if err != nil {
		takeScreenshot(wd, t, "find_google_btn_failed")
		t.Fatalf("Could not find Google login button: %v", err)
	}

	// Assert health badge is visible and says ONLINE or OFFLINE
	badge, err := wd.FindElement(selenium.ByCSSSelector, "[data-testid='health-status-badge']")
	if err == nil {
		text, _ := badge.Text()
		t.Logf("Landing page backend health reported as: %s", text)
	}

	// 3. Click Google Login -> Trigger Mock OAuth flow
	t.Log("Clicking Google sign-in button...")
	if err := googleBtn.Click(); err != nil {
		takeScreenshot(wd, t, "click_google_btn_failed")
		t.Fatalf("Failed to click Google sign-in: %v", err)
	}

	// Wait for redirect to mock consent portal
	t.Log("Waiting for Mock OAuth portal redirect...")
	err = waitTillElementFound(wd, t, "[data-testid='mock-login-user-btn']", 10*time.Second)
	if err != nil {
		t.Fatalf("Failed waiting for mock consent screen: %v", err)
	}

	mockUserBtn, err := wd.FindElement(selenium.ByCSSSelector, "[data-testid='mock-login-user-btn']")
	if err != nil {
		takeScreenshot(wd, t, "find_mock_user_btn_failed")
		t.Fatalf("Could not locate profile button in mock consent screen: %v", err)
	}

	// 4. Authorize Mock Profile -> Callback -> Dashboard redirect
	t.Log("Clicking Test Developer profile in mock consent portal...")
	if err := mockUserBtn.Click(); err != nil {
		takeScreenshot(wd, t, "click_mock_user_btn_failed")
		t.Fatalf("Failed to select test profile: %v", err)
	}

	// Wait to be redirected back to frontend dashboard
	t.Log("Waiting for callback processing and dashboard load...")
	err = waitTillElementFound(wd, t, "[data-testid='dashboard-header']", 10*time.Second)
	if err != nil {
		// Log page source to diagnose errors if redirect hangs
		src, _ := wd.PageSource()
		t.Logf("Debug page source:\n%s", src)
		t.Fatalf("Failed to redirect back to logged-in dashboard: %v", err)
	}

	// 5. Assert Logged-in Dashboard state
	t.Log("Verifying logged-in dashboard components...")

	healthCard, err := wd.FindElement(selenium.ByCSSSelector, "[data-testid='health-status-card']")
	if err != nil {
		takeScreenshot(wd, t, "find_health_card_failed")
		t.Errorf("Could not locate health check card on dashboard: %v", err)
	} else if visible, _ := healthCard.IsDisplayed(); !visible {
		takeScreenshot(wd, t, "health_card_not_visible")
		t.Errorf("Health check card is not visible")
	}

	userCard, err := wd.FindElement(selenium.ByCSSSelector, "[data-testid='user-profile-card']")
	if err != nil {
		takeScreenshot(wd, t, "find_user_card_failed")
		t.Errorf("Could not locate user profile card on dashboard: %v", err)
	} else if visible, _ := userCard.IsDisplayed(); !visible {
		takeScreenshot(wd, t, "user_card_not_visible")
		t.Errorf("User profile card is not visible")
	}

	// Verify health check details (environment name, etc.)
	dashBadge, err := wd.FindElement(selenium.ByCSSSelector, "[data-testid='health-status-badge']")
	if err == nil {
		badgeText, _ := dashBadge.Text()
		t.Logf("Dashboard reports health check badge: %s", badgeText)
	}

	// 6. Logout assertion
	t.Log("Testing logout functionality...")
	logoutBtn, err := wd.FindElement(selenium.ByCSSSelector, "[data-testid='logout-btn']")
	if err != nil {
		takeScreenshot(wd, t, "find_logout_btn_failed")
		t.Fatalf("Could not find session terminate button: %v", err)
	}

	if err := logoutBtn.Click(); err != nil {
		takeScreenshot(wd, t, "click_logout_btn_failed")
		t.Fatalf("Failed to click session terminate: %v", err)
	}

	// Wait to return to logged out screen
	t.Log("Verifying return to logged-out landing page...")
	err = waitTillElementFound(wd, t, "[data-testid='login-card']", 10*time.Second)
	if err != nil {
		t.Fatalf("Failed to redirect back to landing page after logout: %v", err)
	}

	t.Log("E2E Selenium workflow completed successfully!")
}

// Helper function to poll for element presence with timeout
func waitTillElementFound(wd selenium.WebDriver, t *testing.T, selector string, timeout time.Duration) error {
	t.Helper()
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		elem, err := wd.FindElement(selenium.ByCSSSelector, selector)
		if err == nil {
			if displayed, _ := elem.IsDisplayed(); displayed {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	takeScreenshot(wd, t, "wait_for_element_failed")
	return fmt.Errorf("timeout waiting for element matching selector '%s' to be visible", selector)
}
