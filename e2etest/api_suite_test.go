package e2etest

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type APITestSuite struct {
	BaseAPISuite
	backendURL string
}

func (s *APITestSuite) SetupSuite() {
	s.BaseAPISuite.SetupSuite()
	s.backendURL = GlobalConfig.BackendURL
}

func (s *APITestSuite) TestAPIHealthEndpoint() {
	s.CurrentTest.LogStep("Prep Request", "INFO", "Preparing API request for health check endpoint")
	
	req, err := http.NewRequest("GET", s.backendURL+"/health", nil)
	s.Require().NoError(err)

	resp, bodyBytes, err := s.LogAndDoRequest(req, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	s.CurrentTest.LogStep("Decode Response", "INFO", "Decoding health check response body")
	var body map[string]interface{}
	err = json.Unmarshal(bodyBytes, &body)
	s.Require().NoError(err, "Failed to decode health check response JSON")

	// Assertions
	s.CurrentTest.LogStep("Assert Schema", "INFO", "Validating health response schemas and keys")
	status, ok := body["status"].(string)
	s.True(ok, "Expected status as string")
	s.Equal("UP", status)

	env, ok := body["environment"].(string)
	s.True(ok, "Expected environment as string")
	s.NotEmpty(env)

	memory, ok := body["memory"].(map[string]interface{})
	s.True(ok, "Expected memory stats block")
	s.Contains(memory, "alloc_mb")

	deps, ok := body["dependencies"].(map[string]interface{})
	s.True(ok, "Expected dependencies block")
	s.Contains(deps, "google_oauth")
	
	s.CurrentTest.LogStep("Assertions Completed", "PASSED", "All health check fields validated successfully")
}

func (s *APITestSuite) TestAPISwaggerEndpoint() {
	s.CurrentTest.LogStep("Prep HTML Request", "INFO", "Querying Swagger UI HTML page")
	reqHTML, err := http.NewRequest("GET", s.backendURL+"/swagger/index.html", nil)
	s.Require().NoError(err)
	
	respHTML, _, err := s.LogAndDoRequest(reqHTML, nil)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, respHTML.StatusCode)

	s.CurrentTest.LogStep("Prep JSON Request", "INFO", "Querying Swagger JSON API document spec")
	reqJSON, err := http.NewRequest("GET", s.backendURL+"/swagger/doc.json", nil)
	s.Require().NoError(err)

	respJSON, jsonBytes, err := s.LogAndDoRequest(reqJSON, nil)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, respJSON.StatusCode)

	s.CurrentTest.LogStep("Decode Swagger Spec", "INFO", "Decoding Swagger specification document")
	var swagDoc map[string]interface{}
	err = json.Unmarshal(jsonBytes, &swagDoc)
	s.Require().NoError(err, "Failed to decode swagger JSON docs")

	s.CurrentTest.LogStep("Assert Swagger Spec", "INFO", "Validating schema attributes")
	s.Equal("2.0", swagDoc["swagger"])
	
	info, ok := swagDoc["info"].(map[string]interface{})
	s.True(ok, "Expected swagger info block")
	s.Contains(info, "title")

	s.CurrentTest.LogStep("Assertions Completed", "PASSED", "Swagger UI and spec documents successfully verified")
}

func TestRunAPISuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}
