package e2etest

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tebeka/selenium"
)

type WebTestSuite struct {
	BaseWebSuite
	frontendURL string
}

func (s *WebTestSuite) SetupSuite() {
	s.BaseWebSuite.SetupSuite()
	s.frontendURL = GlobalConfig.FrontendURL
}

func (s *WebTestSuite) TestFrontendWorkflow() {
	// 1. Navigate to Frontend
	s.NavigateTo(s.frontendURL)
	s.TakeScreenshot("01_landing_page")

	// 2. Assert Logged Out state
	s.CurrentTest.LogStep("Verify Landing Page State", "INFO", "Validating presence of login card and elements")
	loginCard, err := s.WD.FindElement(selenium.ByCSSSelector, "[data-testid='login-card']")
	s.Require().NoError(err, "Could not find login card. Is frontend running?")
	
	displayed, err := loginCard.IsDisplayed()
	s.Require().NoError(err)
	s.True(displayed, "Login card should be visible")

	_, err = s.WD.FindElement(selenium.ByCSSSelector, "[data-testid='google-login-btn']")
	s.Require().NoError(err, "Google sign-in button missing")

	// Check health status badge
	badge, err := s.WD.FindElement(selenium.ByCSSSelector, "[data-testid='health-status-badge']")
	if err == nil {
		text, _ := badge.Text()
		s.CurrentTest.LogStep("Check Health Status Badge", "PASSED", fmt.Sprintf("Health status badge reports: %s", text))
	} else {
		s.CurrentTest.LogStep("Check Health Status Badge", "INFO", "No health status badge found on landing page")
	}

	// 3. Click Google Login -> Trigger Mock OAuth flow
	s.ClickElement("[data-testid='google-login-btn']", "Google Login Button")
	
	// Wait for redirect to mock consent portal
	s.WaitTillElementFound("[data-testid='mock-login-user-btn']", 10*time.Second)
	s.TakeScreenshot("02_mock_consent_portal")

	// 4. Authorize Mock Profile -> Callback -> Dashboard redirect
	s.ClickElement("[data-testid='mock-login-user-btn']", "Test Developer profile selection button")
	
	// Wait to be redirected back to frontend dashboard
	s.WaitTillElementFound("[data-testid='dashboard-header']", 10*time.Second)
	s.TakeScreenshot("03_dashboard_page")

	// 5. Assert Logged-in Dashboard state
	s.CurrentTest.LogStep("Verify Dashboard Components", "INFO", "Validating presence of cards on dashboard")
	
	healthCard, err := s.WD.FindElement(selenium.ByCSSSelector, "[data-testid='health-status-card']")
	s.Require().NoError(err, "Could not find health check card on dashboard")
	visible, _ := healthCard.IsDisplayed()
	s.True(visible, "Health card should be visible")

	userCard, err := s.WD.FindElement(selenium.ByCSSSelector, "[data-testid='user-profile-card']")
	s.Require().NoError(err, "Could not find user profile card on dashboard")
	visible, _ = userCard.IsDisplayed()
	s.True(visible, "User profile card should be visible")

	dashBadge, err := s.WD.FindElement(selenium.ByCSSSelector, "[data-testid='health-status-badge']")
	if err == nil {
		text, _ := dashBadge.Text()
		s.CurrentTest.LogStep("Check Dashboard Health Badge", "PASSED", fmt.Sprintf("Dashboard reports backend status: %s", text))
	}

	// 6. Logout assertion
	s.ClickElement("[data-testid='logout-btn']", "Logout Button")

	// Wait to return to logged out screen
	s.WaitTillElementFound("[data-testid='login-card']", 10*time.Second)
	s.TakeScreenshot("04_returned_to_landing")

	s.CurrentTest.LogStep("Workflow Complete", "PASSED", "Successfully traversed login, mock authorization, dashboard verification, and logout workflow")
}

func TestRunWebSuite(t *testing.T) {
	suite.Run(t, new(WebTestSuite))
}
