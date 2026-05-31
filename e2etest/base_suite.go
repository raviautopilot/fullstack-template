package e2etest

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tebeka/selenium"
)

// BaseAPISuite provides standardized hooks and logging helpers for API testing
type BaseAPISuite struct {
	suite.Suite
	SuiteReport *TestSuite
	CurrentTest *TestCase
}

func (s *BaseAPISuite) SetupSuite() {
	s.SuiteReport = GetReport().AddSuite("API E2E Tests")
}

func (s *BaseAPISuite) SetupTest() {
	s.CurrentTest = s.SuiteReport.AddTestCase(s.T().Name())
}

func (s *BaseAPISuite) TearDownTest() {
	s.CurrentTest.Duration = time.Since(s.CurrentTest.StartTime)
	if s.T().Failed() {
		s.CurrentTest.Status = "FAILED"
		if s.CurrentTest.ErrorMsg == "" {
			s.CurrentTest.ErrorMsg = "Test failed with assertion errors"
		}
	}
}

// LogAndDoRequest executes an HTTP request, automatically records it in the report, and returns the response body
func (s *BaseAPISuite) LogAndDoRequest(req *http.Request, reqBody []byte) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	s.T().Logf("API Call: %s %s", req.Method, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		s.CurrentTest.LogStep("API Request Failed", "FAILED", err.Error())
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.CurrentTest.LogStep("API Read Response Body Failed", "FAILED", err.Error())
		return resp, nil, err
	}

	status := "PASSED"
	if resp.StatusCode >= 400 {
		status = "FAILED"
	}
	s.CurrentTest.LogAPIStep("API Call Completed", status, req, reqBody, resp, respBody)
	return resp, respBody, nil
}

// BaseWebSuite provides standardized hooks, driver controls, and action logs for Web E2E testing
type BaseWebSuite struct {
	suite.Suite
	SuiteReport      *TestSuite
	CurrentTest      *TestCase
	SeleniumService  *selenium.Service
	WD               selenium.WebDriver
	ChromeDriverPort int
}

func (s *BaseWebSuite) SetupSuite() {
	s.SuiteReport = GetReport().AddSuite("Web UI E2E Tests")
	s.ChromeDriverPort = 8082

	var chromeDriverPath = os.Getenv("CHROMEDRIVER_PATH")
	if chromeDriverPath == "" {
		chromeDriverPath = "/usr/bin/chromedriver"
	}

	opts := []selenium.ServiceOption{
		selenium.Output(os.Stderr),
	}

	s.T().Logf("Starting ChromeDriver service on port %d...", s.ChromeDriverPort)
	service, err := selenium.NewChromeDriverService(chromeDriverPath, s.ChromeDriverPort, opts...)
	if err != nil {
		s.T().Fatalf("Failed to start ChromeDriver service: %v", err)
	}
	s.SeleniumService = service
}

func (s *BaseWebSuite) TearDownSuite() {
	if s.SeleniumService != nil {
		s.SeleniumService.Stop()
	}
}

func (s *BaseWebSuite) SetupTest() {
	s.CurrentTest = s.SuiteReport.AddTestCase(s.T().Name())

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

	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", s.ChromeDriverPort))
	if err != nil {
		s.CurrentTest.LogStep("Remote Driver Session Init", "FAILED", err.Error())
		s.T().Fatalf("Failed to open remote Selenium session: %v", err)
	}
	s.WD = wd

	if err := wd.ResizeWindow("current", 1280, 800); err != nil {
		s.CurrentTest.LogStep("Resize Window", "INFO", fmt.Sprintf("Warning: failed to resize browser window: %v", err))
	}
}

func (s *BaseWebSuite) TearDownTest() {
	s.CurrentTest.Duration = time.Since(s.CurrentTest.StartTime)
	if s.T().Failed() {
		s.CurrentTest.Status = "FAILED"
		s.TakeScreenshot("failure_state")
	}

	if s.WD != nil {
		s.WD.Quit()
	}
}

// TakeScreenshot captures browser state, saves it to reports_evidences directory, and adds screenshot path to the active report step
func (s *BaseWebSuite) TakeScreenshot(name string) {
	if s.WD == nil {
		return
	}
	screenshot, err := s.WD.Screenshot()
	if err != nil {
		s.CurrentTest.LogStep("Take Screenshot", "INFO", fmt.Sprintf("Failed to take screenshot: %v", err))
		return
	}

	evidencesDir := "report_evidences"
	if _, err := os.Stat(evidencesDir); os.IsNotExist(err) {
		os.MkdirAll(evidencesDir, os.ModePerm)
	}

	filename := fmt.Sprintf("%s_%s.png", name, time.Now().Format("150405_000"))
	fullPath := filepath.Join(evidencesDir, filename)
	if err := os.WriteFile(fullPath, screenshot, 0644); err != nil {
		s.CurrentTest.LogStep("Save Screenshot", "INFO", fmt.Sprintf("Failed to save screenshot: %v", err))
	} else {
		// Use relative path for HTML report
		relPath := filepath.Join("report_evidences", filename)
		s.CurrentTest.LogScreenshotStep("Capture Evidence", "INFO", fmt.Sprintf("Screenshot saved: %s", relPath), relPath)
	}
}

// NavigateTo loads a URL and logs navigation details
func (s *BaseWebSuite) NavigateTo(url string) {
	s.CurrentTest.LogStep("Navigation", "INFO", fmt.Sprintf("Navigating to URL: %s", url))
	if err := s.WD.Get(url); err != nil {
		s.CurrentTest.LogStep("Navigation Failed", "FAILED", err.Error())
		s.T().Fatalf("Failed to load page: %v", err)
	}
	// Small sleep after navigation
	time.Sleep(2 * time.Second)
}

// ClickElement clicks on a CSS selector and logs it
func (s *BaseWebSuite) ClickElement(selector, desc string) {
	s.CurrentTest.LogStep("Click Element", "INFO", fmt.Sprintf("Clicking %s (%s)", selector, desc))
	elem, err := s.WD.FindElement(selenium.ByCSSSelector, selector)
	if err != nil {
		s.CurrentTest.LogStep("Find Click Target Failed", "FAILED", fmt.Sprintf("Could not find element: %s. Error: %v", selector, err))
		s.T().Fatalf("Click element failed: %v", err)
	}

	if err := elem.Click(); err != nil {
		s.CurrentTest.LogStep("Click Execution Failed", "FAILED", fmt.Sprintf("Could not click element: %s. Error: %v", selector, err))
		s.T().Fatalf("Click element failed: %v", err)
	}
}

// WaitTillElementFound waits until an element is visible and active on the DOM, logging intermediate steps
func (s *BaseWebSuite) WaitTillElementFound(selector string, timeout time.Duration) {
	s.CurrentTest.LogStep("Wait For Element", "INFO", fmt.Sprintf("Waiting for element visible: %s", selector))
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		elem, err := s.WD.FindElement(selenium.ByCSSSelector, selector)
		if err == nil {
			if displayed, _ := elem.IsDisplayed(); displayed {
				s.CurrentTest.LogStep("Element Located", "PASSED", fmt.Sprintf("Located visible element: %s", selector))
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}

	s.CurrentTest.LogStep("Wait Timeout", "FAILED", fmt.Sprintf("Timeout waiting for element matching selector '%s' to be visible", selector))
	s.T().Fatalf("Timeout waiting for element '%s'", selector)
}
