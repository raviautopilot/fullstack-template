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
	LogExecution("API SUITE: Initializing E2E API Test Suite...")
	s.SuiteReport = GetReport().AddSuite("API E2E Tests")
}

func (s *BaseAPISuite) SetupTest() {
	LogExecution("API TEST [%s]: Setting up test context...", s.T().Name())
	s.CurrentTest = s.SuiteReport.AddTestCase(s.T().Name())
}

func (s *BaseAPISuite) TearDownTest() {
	s.CurrentTest.Duration = time.Since(s.CurrentTest.StartTime)
	if s.T().Failed() {
		s.CurrentTest.Status = "FAILED"
		if s.CurrentTest.ErrorMsg == "" {
			s.CurrentTest.ErrorMsg = "Test failed with assertion errors"
		}
		LogExecution("API TEST [%s] FAILED | Error: %s", s.T().Name(), s.CurrentTest.ErrorMsg)
	} else {
		LogExecution("API TEST [%s] PASSED | Duration: %v", s.T().Name(), s.CurrentTest.Duration)
	}
}

// LogAndDoRequest executes an HTTP request, automatically records it in the report, and returns the response body
func (s *BaseAPISuite) LogAndDoRequest(req *http.Request, reqBody []byte) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	LogExecution("API TEST [%s] CALL: %s %s", s.T().Name(), req.Method, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		s.CurrentTest.LogStep("API Request Failed", "FAILED", err.Error())
		LogExecution("API TEST [%s] CALL ERROR: %v", s.T().Name(), err)
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.CurrentTest.LogStep("API Read Response Body Failed", "FAILED", err.Error())
		LogExecution("API TEST [%s] BODY READ ERROR: %v", s.T().Name(), err)
		return resp, nil, err
	}

	status := "PASSED"
	if resp.StatusCode >= 400 {
		status = "FAILED"
	}

	LogExecution("API TEST [%s] CALL COMPLETED | Response Status: %s", s.T().Name(), resp.Status)

	// Only record detailed API dumps if EnableEvidence is true in config
	if GlobalConfig.EnableEvidence {
		s.CurrentTest.LogAPIStep("API Call Completed", status, req, reqBody, resp, respBody)
	} else {
		s.CurrentTest.LogStep("API Call Completed", status, fmt.Sprintf("API Call: %s %s -> Status: %s (payload dumps disabled)", req.Method, req.URL.Path, resp.Status))
	}
	
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
	LogExecution("WEB SUITE: Initializing E2E Web UI Test Suite...")
	s.SuiteReport = GetReport().AddSuite("Web UI E2E Tests")
	s.ChromeDriverPort = GlobalConfig.ChromeDriverPort

	chromeDriverPath := GlobalConfig.ChromeDriverPath
	
	// Redirect raw ChromeDriver debugger outputs to deep-debug.log to avoid console pollution
	var chromeDriverOutput io.Writer = os.Stderr
	if DeepDebugWriter != nil {
		chromeDriverOutput = DeepDebugWriter
	}
	opts := []selenium.ServiceOption{
		selenium.Output(chromeDriverOutput),
	}

	LogExecution("WEB SUITE: Starting ChromeDriver service on port %d...", s.ChromeDriverPort)
	service, err := selenium.NewChromeDriverService(chromeDriverPath, s.ChromeDriverPort, opts...)
	if err != nil {
		LogExecution("WEB SUITE ERROR: Failed to launch ChromeDriver: %v", err)
		s.T().Fatalf("Failed to start ChromeDriver service: %v", err)
	}
	s.SeleniumService = service
}

func (s *BaseWebSuite) TearDownSuite() {
	LogExecution("WEB SUITE: Stopping ChromeDriver service...")
	if s.SeleniumService != nil {
		s.SeleniumService.Stop()
	}
}

func (s *BaseWebSuite) SetupTest() {
	LogExecution("WEB TEST [%s]: Setting up test context...", s.T().Name())
	s.CurrentTest = s.SuiteReport.AddTestCase(s.T().Name())

	caps := selenium.Capabilities{"browserName": "chrome"}
	chromiumPath := GlobalConfig.ChromiumPath

	args := []string{
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--remote-debugging-port=9222",
	}
	if GlobalConfig.Headless {
		args = append(args, "--headless=new")
	}

	chromeCaps := map[string]interface{}{
		"binary": chromiumPath,
		"args":   args,
	}
	caps["goog:chromeOptions"] = chromeCaps

	LogExecution("WEB TEST [%s]: Opening ChromeDriver WebDriver remote session (Headless: %t)...", s.T().Name(), GlobalConfig.Headless)
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", s.ChromeDriverPort))
	if err != nil {
		s.CurrentTest.LogStep("Remote Driver Session Init", "FAILED", err.Error())
		LogExecution("WEB TEST [%s] WEBDRIVER ERROR: %v", s.T().Name(), err)
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
		LogExecution("WEB TEST [%s] FAILED | Capturing failure screenshot...", s.T().Name())
		s.TakeScreenshot("failure_state")
	} else {
		LogExecution("WEB TEST [%s] PASSED | Duration: %v", s.T().Name(), s.CurrentTest.Duration)
	}

	LogExecution("WEB TEST [%s]: Closing Chrome browser session...", s.T().Name())
	if s.WD != nil {
		s.WD.Quit()
	}
}

// TakeScreenshot captures browser state, saves it to reports_evidences directory, and adds screenshot path to the active report step
func (s *BaseWebSuite) TakeScreenshot(name string) {
	if s.WD == nil {
		return
	}

	// Skip taking screenshots if EnableEvidence is false in config
	if !GlobalConfig.EnableEvidence {
		s.CurrentTest.LogStep("Take Screenshot", "INFO", "Skipped screenshot (evidence disabled in config)")
		LogExecution("WEB TEST [%s]: Skipped screenshot capture (evidence disabled in config)", s.T().Name())
		return
	}

	LogExecution("WEB TEST [%s]: Capturing visual evidence screenshot for '%s'...", s.T().Name(), name)
	screenshot, err := s.WD.Screenshot()
	if err != nil {
		s.CurrentTest.LogStep("Take Screenshot", "INFO", fmt.Sprintf("Failed to take screenshot: %v", err))
		LogExecution("WEB TEST [%s] SCREENSHOT ERROR: %v", s.T().Name(), err)
		return
	}

	runDir := GetReport().GetRunDirectory()
	evidencesDir := filepath.Join(runDir, GlobalConfig.EvidenceDir)
	if _, err := os.Stat(evidencesDir); os.IsNotExist(err) {
		os.MkdirAll(evidencesDir, os.ModePerm)
	}

	filename := fmt.Sprintf("%s.png", name)
	fullPath := filepath.Join(evidencesDir, filename)
	if err := os.WriteFile(fullPath, screenshot, 0644); err != nil {
		s.CurrentTest.LogStep("Save Screenshot", "INFO", fmt.Sprintf("Failed to save screenshot: %v", err))
		LogExecution("WEB TEST [%s] SCREENSHOT SAVE ERROR: %v", s.T().Name(), err)
	} else {
		// Use relative path for HTML report (relative to report.html in runDir)
		relPath := filepath.Join(GlobalConfig.EvidenceDir, filename)
		s.CurrentTest.LogScreenshotStep("Capture Evidence", "INFO", fmt.Sprintf("Screenshot saved: %s", relPath), relPath)
		LogExecution("WEB TEST [%s]: Visual evidence saved successfully to '%s'", s.T().Name(), fullPath)
	}
}

// NavigateTo loads a URL and logs navigation details
func (s *BaseWebSuite) NavigateTo(url string) {
	LogExecution("WEB TEST [%s]: Navigating browser to URL: %s", s.T().Name(), url)
	s.CurrentTest.LogStep("Navigation", "INFO", fmt.Sprintf("Navigating to URL: %s", url))
	if err := s.WD.Get(url); err != nil {
		s.CurrentTest.LogStep("Navigation Failed", "FAILED", err.Error())
		LogExecution("WEB TEST [%s] NAVIGATION ERROR: %v", s.T().Name(), err)
		s.T().Fatalf("Failed to load page: %v", err)
	}
	// Small sleep after navigation
	time.Sleep(2 * time.Second)
}

// ClickElement clicks on a CSS selector and logs it
func (s *BaseWebSuite) ClickElement(selector, desc string) {
	LogExecution("WEB TEST [%s]: Clicking element matching selector '%s' (%s)", s.T().Name(), selector, desc)
	s.CurrentTest.LogStep("Click Element", "INFO", fmt.Sprintf("Clicking %s (%s)", selector, desc))
	elem, err := s.WD.FindElement(selenium.ByCSSSelector, selector)
	if err != nil {
		s.CurrentTest.LogStep("Find Click Target Failed", "FAILED", fmt.Sprintf("Could not find element: %s. Error: %v", selector, err))
		LogExecution("WEB TEST [%s] FIND ELEMENT ERROR (selector: %s): %v", s.T().Name(), selector, err)
		s.T().Fatalf("Click element failed: %v", err)
	}

	if err := elem.Click(); err != nil {
		s.CurrentTest.LogStep("Click Execution Failed", "FAILED", fmt.Sprintf("Could not click element: %s. Error: %v", selector, err))
		LogExecution("WEB TEST [%s] CLICK ERROR (selector: %s): %v", s.T().Name(), selector, err)
		s.T().Fatalf("Click element failed: %v", err)
	}
}

// WaitTillElementFound waits until an element is visible and active on the DOM, logging intermediate steps
func (s *BaseWebSuite) WaitTillElementFound(selector string, timeout time.Duration) {
	LogExecution("WEB TEST [%s]: Waiting up to %v for element matching selector '%s' to become visible...", s.T().Name(), timeout, selector)
	s.CurrentTest.LogStep("Wait For Element", "INFO", fmt.Sprintf("Waiting for element visible: %s", selector))
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		elem, err := s.WD.FindElement(selenium.ByCSSSelector, selector)
		if err == nil {
			if displayed, _ := elem.IsDisplayed(); displayed {
				s.CurrentTest.LogStep("Element Located", "PASSED", fmt.Sprintf("Located visible element: %s", selector))
				LogExecution("WEB TEST [%s]: Element successfully located matching selector '%s'", s.T().Name(), selector)
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}

	s.CurrentTest.LogStep("Wait Timeout", "FAILED", fmt.Sprintf("Timeout waiting for element matching selector '%s' to be visible", selector))
	LogExecution("WEB TEST [%s] WAIT TIMEOUT: Element matching selector '%s' not visible after %v", s.T().Name(), selector, timeout)
	s.T().Fatalf("Timeout waiting for element '%s'", selector)
}
