package e2etest

import (
	"os"
	"path/filepath"
	"testing"
)

// GlobalConfig holds test fixtures and options loaded from e2e-test.json
var GlobalConfig *Config

func TestMain(m *testing.M) {
	// Load config fixtures from e2e-test.json (generates sample if not present)
	var err error
	GlobalConfig, err = LoadConfig("e2e-test.json")
	if err != nil {
		println("Error loading config e2e-test.json: ", err.Error())
		os.Exit(1)
	}

	// Execute all registered test suites in the package
	exitCode := m.Run()

	// Finalize the global test report and write it as an interactive dashboard
	report := GetReport()
	report.Finalize()

	runDir := report.GetRunDirectory()
	reportFilePath := filepath.Join(runDir, GlobalConfig.ReportPath)

	if err := report.GenerateHTML(reportFilePath); err != nil {
		println("Error generating E2E HTML test report: ", err.Error())
	} else {
		println("Interactive E2E test report generated successfully: ", reportFilePath)
	}

	os.Exit(exitCode)
}
