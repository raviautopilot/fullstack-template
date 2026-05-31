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

	// Finalize the global test report and write it as an interactive dashboard
	report := GetReport()
	runDir := report.GetRunDirectory()
	if err := os.MkdirAll(runDir, 0755); err != nil {
		println("Error creating run directory: ", err.Error())
		os.Exit(1)
	}

	// Open execution.log
	execLogFile, err := os.OpenFile(filepath.Join(runDir, "execution.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		ExecutionLogWriter = execLogFile
		defer execLogFile.Close()
	} else {
		println("Error creating execution.log: ", err.Error())
	}

	// Open deep-debug.log
	deepDebugFile, err := os.OpenFile(filepath.Join(runDir, "deep-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		DeepDebugWriter = deepDebugFile
		defer deepDebugFile.Close()
	} else {
		println("Error creating deep-debug.log: ", err.Error())
	}

	// Log E2E run start header
	LogExecution("================================================================================")
	LogExecution("E2E SUITE RUN STARTED | Time: %s", report.StartTime.Format("2006-01-02 15:04:05"))
	LogExecution("HEADLESS MODE: %t | EVIDENCE LOGGING: %t", GlobalConfig.Headless, GlobalConfig.EnableEvidence)
	LogExecution("================================================================================")

	// Execute all registered test suites in the package
	exitCode := m.Run()

	report.Finalize()

	// Log E2E run summary footer
	LogExecution("================================================================================")
	LogExecution("E2E SUITE RUN COMPLETED | Total Duration: %v", report.Duration)
	LogExecution("STATS: Total Tests = %d | Passed = %d | Failed = %d", report.TotalTests, report.PassedTests, report.FailedTests)
	LogExecution("================================================================================")

	reportFilePath := filepath.Join(runDir, GlobalConfig.ReportPath)
	if err := report.GenerateHTML(reportFilePath); err != nil {
		println("Error generating E2E HTML test report: ", err.Error())
	} else {
		println("Interactive E2E test report generated successfully: ", reportFilePath)
	}

	os.Exit(exitCode)
}
