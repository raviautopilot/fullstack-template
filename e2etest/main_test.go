package e2etest

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Execute all registered test suites in the package
	exitCode := m.Run()

	// Finalize the global test report and write it as an interactive dashboard
	report := GetReport()
	report.Finalize()

	outputPath := "report.html"
	if err := report.GenerateHTML(outputPath); err != nil {
		println("Error generating E2E HTML test report: ", err.Error())
	} else {
		println("Interactive E2E test report generated successfully: ", outputPath)
	}

	os.Exit(exitCode)
}
