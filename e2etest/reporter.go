package e2etest

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type APIRequestDump struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

type APIResponseDump struct {
	Status     string              `json:"status"`
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
}

type Step struct {
	Timestamp   time.Time        `json:"timestamp"`
	Name        string           `json:"name"`
	Status      string           `json:"status"` // "PASSED", "FAILED", "INFO"
	Message     string           `json:"message"`
	Screenshot  string           `json:"screenshot,omitempty"` // Relative path to screenshot
	APIRequest  *APIRequestDump  `json:"api_request,omitempty"`
	APIResponse *APIResponseDump `json:"api_response,omitempty"`
}

type TestCase struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"` // "PASSED", "FAILED"
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	Steps     []Step        `json:"steps"`
	ErrorMsg  string        `json:"error_msg,omitempty"`
	mu        sync.Mutex
}

type TestSuite struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"` // "PASSED", "FAILED"
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	TestCases []*TestCase   `json:"test_cases"`
	mu        sync.Mutex
}

type Report struct {
	ProjectName string        `json:"project_name"`
	StartTime   time.Time     `json:"start_time"`
	Duration    time.Duration `json:"duration"`
	Suites      []*TestSuite  `json:"suites"`
	TotalTests  int           `json:"total_tests"`
	PassedTests int           `json:"passed_tests"`
	FailedTests int           `json:"failed_tests"`
	mu          sync.Mutex
}

var (
	globalReport *Report
	reportOnce   sync.Once
)

// GetReport returns the singleton instance of the global report
func GetReport() *Report {
	reportOnce.Do(func() {
		globalReport = &Report{
			ProjectName: "E2E Test Suite (Web & API)",
			StartTime:   time.Now(),
			Suites:      make([]*TestSuite, 0),
		}
	})
	return globalReport
}

// GetRunDirectory returns the active time-stamped directory for this test execution run
func (r *Report) GetRunDirectory() string {
	return filepath.Join("reports", "run_"+r.StartTime.Format("2006-01-02_15-04-05"))
}

var (
	ExecutionLogWriter io.Writer
	DeepDebugWriter    io.Writer
)

// LogExecution logs a high-level test step to Stdout and the execution.log file
func LogExecution(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	formattedMsg := fmt.Sprintf("[%s] %s\n", timestamp, msg)

	// Output to terminal Stdout
	fmt.Print(formattedMsg)

	// Output to execution.log
	if ExecutionLogWriter != nil {
		ExecutionLogWriter.Write([]byte(formattedMsg))
	}
}

// AddSuite creates and adds a new test suite
func (r *Report) AddSuite(name string) *TestSuite {
	r.mu.Lock()
	defer r.mu.Unlock()

	suite := &TestSuite{
		Name:      name,
		Status:    "PASSED",
		StartTime: time.Now(),
		TestCases: make([]*TestCase, 0),
	}
	r.Suites = append(r.Suites, suite)
	return suite
}

// AddTestCase creates and adds a test case to a suite
func (s *TestSuite) AddTestCase(name string) *TestCase {
	s.mu.Lock()
	defer s.mu.Unlock()

	tc := &TestCase{
		Name:      name,
		Status:    "PASSED",
		StartTime: time.Now(),
		Steps:     make([]Step, 0),
	}
	s.TestCases = append(s.TestCases, tc)
	return tc
}

// LogStep logs a standard step in a test case
func (tc *TestCase) LogStep(name, status, message string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.Steps = append(tc.Steps, Step{
		Timestamp: time.Now(),
		Name:      name,
		Status:    status,
		Message:   message,
	})
	if status == "FAILED" {
		tc.Status = "FAILED"
		tc.ErrorMsg = message
	}
}

// LogScreenshotStep logs a step with a screenshot path
func (tc *TestCase) LogScreenshotStep(name, status, message, screenshotPath string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.Steps = append(tc.Steps, Step{
		Timestamp:  time.Now(),
		Name:       name,
		Status:     status,
		Message:    message,
		Screenshot: screenshotPath,
	})
	if status == "FAILED" {
		tc.Status = "FAILED"
		tc.ErrorMsg = message
	}
}

// LogAPIStep logs an API request and response
func (tc *TestCase) LogAPIStep(name, status string, req *http.Request, reqBody []byte, resp *http.Response, respBody []byte) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	var reqDump *APIRequestDump
	if req != nil {
		reqDump = &APIRequestDump{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: req.Header,
			Body:    string(reqBody),
		}
	}

	var respDump *APIResponseDump
	if resp != nil {
		respDump = &APIResponseDump{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       string(respBody),
		}
	}

	tc.Steps = append(tc.Steps, Step{
		Timestamp:   time.Now(),
		Name:        name,
		Status:      status,
		Message:     fmt.Sprintf("API Call: %s %s -> Status: %s", req.Method, req.URL.Path, resp.Status),
		APIRequest:  reqDump,
		APIResponse: respDump,
	})

	if status == "FAILED" {
		tc.Status = "FAILED"
	}
}

// Finalize calculates stats and duration for the report
func (r *Report) Finalize() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Duration = time.Since(r.StartTime)
	r.TotalTests = 0
	r.PassedTests = 0
	r.FailedTests = 0

	for _, suite := range r.Suites {
		suite.mu.Lock()
		suite.Duration = time.Since(suite.StartTime)
		suiteFailed := false

		for _, tc := range suite.TestCases {
			tc.mu.Lock()
			if tc.Duration == 0 {
				tc.Duration = time.Since(tc.StartTime)
			}
			r.TotalTests++
			if tc.Status == "FAILED" {
				r.FailedTests++
				suiteFailed = true
			} else {
				r.PassedTests++
			}
			tc.mu.Unlock()
		}

		if suiteFailed {
			suite.Status = "FAILED"
		} else {
			suite.Status = "PASSED"
		}
		suite.mu.Unlock()
	}
}

// GenerateHTML generates a premium HTML report file
func (r *Report) GenerateHTML(outputPath string) error {
	r.Finalize()

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"formatDuration": func(d time.Duration) string {
			if d < time.Millisecond {
				return fmt.Sprintf("%d \u00b5s", d.Microseconds())
			}
			if d < time.Second {
				return fmt.Sprintf("%d ms", d.Milliseconds())
			}
			return fmt.Sprintf("%.2f s", d.Seconds())
		},
		"formatTime": func(t time.Time) string {
			return t.Format("15:04:05.000")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"percentage": func(passed, total int) string {
			if total == 0 {
				return "0.0%"
			}
			return fmt.Sprintf("%.1f%%", float64(passed)/float64(total)*100)
		},
		"jsonPretty": func(v interface{}) string {
			if v == nil {
				return ""
			}
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(b)
		},
	}).Parse(htmlTemplate)

	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, r)
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.ProjectName}} - Test Report</title>
    <style>
        :root {
            --bg-primary: #0f172a;
            --bg-secondary: #1e293b;
            --bg-tertiary: #334155;
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --accent-success: #10b981;
            --accent-failure: #ef4444;
            --accent-info: #3b82f6;
            --accent-warning: #f59e0b;
            --border-color: #475569;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Inter', system-ui, -apple-system, sans-serif;
            background-color: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.5;
            padding: 2rem;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
        }

        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding-bottom: 2rem;
            border-bottom: 1px solid var(--border-color);
            margin-bottom: 2rem;
        }

        h1 {
            font-size: 2.25rem;
            font-weight: 800;
            background: linear-gradient(to right, #60a5fa, #34d399);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .subtitle {
            color: var(--text-secondary);
            font-size: 0.875rem;
            margin-top: 0.25rem;
        }

        /* Dashboard Overview Grid */
        .dashboard-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2.5rem;
        }

        .stat-card {
            background-color: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 0.75rem;
            padding: 1.5rem;
            text-align: center;
            box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
            transition: transform 0.2s;
        }

        .stat-card:hover {
            transform: translateY(-2px);
        }

        .stat-label {
            color: var(--text-secondary);
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 0.5rem;
            font-weight: 700;
        }

        .stat-val {
            font-size: 2rem;
            font-weight: 800;
        }

        .stat-val.passed { color: var(--accent-success); }
        .stat-val.failed { color: var(--accent-failure); }
        .stat-val.duration { color: var(--accent-info); }

        /* Filter buttons */
        .filter-container {
            display: flex;
            gap: 0.75rem;
            margin-bottom: 1.5rem;
        }

        .btn-filter {
            background-color: var(--bg-secondary);
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            padding: 0.5rem 1rem;
            border-radius: 0.375rem;
            cursor: pointer;
            font-size: 0.875rem;
            font-weight: 600;
            transition: all 0.2s;
        }

        .btn-filter:hover {
            background-color: var(--bg-tertiary);
        }

        .btn-filter.active {
            background-color: var(--accent-info);
            border-color: var(--accent-info);
        }

        /* Suite Section */
        .suite-section {
            background-color: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 0.75rem;
            margin-bottom: 2rem;
            overflow: hidden;
            box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
        }

        .suite-header {
            padding: 1.25rem 1.5rem;
            background-color: rgba(255, 255, 255, 0.02);
            border-bottom: 1px solid var(--border-color);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .suite-title {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            font-size: 1.25rem;
            font-weight: 700;
        }

        .badge {
            font-size: 0.75rem;
            padding: 0.25rem 0.5rem;
            border-radius: 9999px;
            font-weight: 700;
            text-transform: uppercase;
        }

        .badge.passed {
            background-color: rgba(16, 185, 129, 0.2);
            color: var(--accent-success);
            border: 1px solid var(--accent-success);
        }

        .badge.failed {
            background-color: rgba(239, 68, 68, 0.2);
            color: var(--accent-failure);
            border: 1px solid var(--accent-failure);
        }

        .suite-meta {
            font-size: 0.875rem;
            color: var(--text-secondary);
        }

        /* Test Cases */
        .test-case {
            border-bottom: 1px solid var(--border-color);
            transition: background-color 0.2s;
        }

        .test-case:last-child {
            border-bottom: none;
        }

        .test-case-header {
            padding: 1rem 1.5rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            cursor: pointer;
            user-select: none;
        }

        .test-case-header:hover {
            background-color: rgba(255, 255, 255, 0.02);
        }

        .test-case-title {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            font-weight: 600;
        }

        .test-case-meta {
            display: flex;
            align-items: center;
            gap: 1rem;
            font-size: 0.875rem;
            color: var(--text-secondary);
        }

        .chevron {
            transition: transform 0.2s;
        }

        .expanded .chevron {
            transform: rotate(90deg);
        }

        /* Test Case Details */
        .test-case-body {
            display: none;
            padding: 1.5rem;
            background-color: rgba(0, 0, 0, 0.15);
            border-top: 1px solid var(--border-color);
        }

        .expanded + .test-case-body {
            display: block;
        }

        /* Timelines / Steps */
        .timeline {
            display: flex;
            flex-direction: column;
            gap: 1.25rem;
            position: relative;
            padding-left: 1.5rem;
        }

        .timeline::before {
            content: '';
            position: absolute;
            left: 5px;
            top: 8px;
            bottom: 8px;
            width: 2px;
            background-color: var(--border-color);
        }

        .step-item {
            position: relative;
        }

        .step-dot {
            position: absolute;
            left: -22px;
            top: 6px;
            width: 12px;
            height: 12px;
            border-radius: 9999px;
            background-color: var(--border-color);
            border: 2px solid var(--bg-secondary);
        }

        .step-dot.passed { background-color: var(--accent-success); }
        .step-dot.failed { background-color: var(--accent-failure); }
        .step-dot.info { background-color: var(--accent-info); }

        .step-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 0.25rem;
        }

        .step-name {
            font-weight: 600;
            font-size: 0.95rem;
        }

        .step-time {
            font-size: 0.75rem;
            color: var(--text-secondary);
        }

        .step-message {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-bottom: 0.5rem;
        }

        /* Screenshot visualizer */
        .screenshot-container {
            margin-top: 0.5rem;
            border: 1px solid var(--border-color);
            border-radius: 0.375rem;
            max-width: 500px;
            overflow: hidden;
            background-color: #000;
            cursor: zoom-in;
        }

        .screenshot-container img {
            width: 100%;
            height: auto;
            display: block;
            transition: transform 0.2s;
        }

        .screenshot-container img:hover {
            transform: scale(1.02);
        }

        /* API Inspector */
        .api-dump {
            margin-top: 0.75rem;
            border: 1px solid var(--border-color);
            border-radius: 0.5rem;
            overflow: hidden;
            font-size: 0.8125rem;
        }

        .api-header {
            background-color: var(--bg-tertiary);
            padding: 0.5rem 1rem;
            font-weight: 700;
            font-family: monospace;
            display: flex;
            justify-content: space-between;
            align-items: center;
            cursor: pointer;
        }

        .api-details {
            background-color: #0c1017;
            padding: 1rem;
            display: flex;
            flex-direction: column;
            gap: 1rem;
            border-top: 1px solid var(--border-color);
        }

        .api-section-title {
            color: #60a5fa;
            font-weight: bold;
            margin-bottom: 0.25rem;
            text-transform: uppercase;
            font-size: 0.75rem;
            letter-spacing: 0.05em;
        }

        pre {
            font-family: 'JetBrains Mono', 'Fira Code', monospace;
            color: #c9d1d9;
            background-color: #161b22;
            padding: 0.75rem;
            border-radius: 0.25rem;
            overflow-x: auto;
            white-space: pre-wrap;
            border: 1px solid #21262d;
        }

        .http-method {
            padding: 0.125rem 0.375rem;
            border-radius: 0.25rem;
            color: #fff;
            font-weight: 800;
            font-size: 0.75rem;
        }
        .http-method.GET { background-color: #059669; }
        .http-method.POST { background-color: #2563eb; }
        .http-method.PUT { background-color: #d97706; }
        .http-method.DELETE { background-color: #dc2626; }

        /* Modal for full screenshot view */
        .modal {
            display: none;
            position: fixed;
            z-index: 9999;
            top: 0;
            left: 0;
            width: 100vw;
            height: 100vh;
            background-color: rgba(0,0,0,0.9);
            justify-content: center;
            align-items: center;
            cursor: zoom-out;
        }

        .modal img {
            max-width: 90%;
            max-height: 90%;
            border: 2px solid var(--border-color);
            box-shadow: 0 25px 50px -12px rgba(0,0,0,0.5);
            border-radius: 0.5rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div>
                <h1>{{.ProjectName}}</h1>
                <div class="subtitle">Generated on {{formatDateTime .StartTime}} | E2E Suite Run</div>
            </div>
            <div style="text-align: right">
                <div class="badge passed" style="font-size: 0.875rem">Success Rate: {{percentage .PassedTests .TotalTests}}</div>
            </div>
        </header>

        <!-- Stats Grid -->
        <section class="dashboard-grid">
            <div class="stat-card">
                <div class="stat-label">Total Tests</div>
                <div class="stat-val">{{.TotalTests}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Passed</div>
                <div class="stat-val passed">{{.PassedTests}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Failed</div>
                <div class="stat-val failed">{{.FailedTests}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Duration</div>
                <div class="stat-val duration">{{formatDuration .Duration}}</div>
            </div>
        </section>

        <!-- Filters -->
        <div class="filter-container">
            <button class="btn-filter active" onclick="filterTests('all')">All Tests</button>
            <button class="btn-filter" onclick="filterTests('PASSED')">Passed ({{.PassedTests}})</button>
            <button class="btn-filter" onclick="filterTests('FAILED')">Failed ({{.FailedTests}})</button>
        </div>

        <!-- Suites -->
        {{range .Suites}}
        <section class="suite-section" data-suite-status="{{.Status}}">
            <div class="suite-header">
                <div class="suite-title">
                    <span class="badge {{if eq .Status "PASSED"}}passed{{else}}failed{{end}}">{{.Status}}</span>
                    <span>{{.Name}}</span>
                </div>
                <div class="suite-meta">
                    <span>{{len .TestCases}} Tests</span> &bull; <span>Duration: {{formatDuration .Duration}}</span>
                </div>
            </div>

            <!-- Test Cases -->
            <div>
                {{range .TestCases}}
                <div class="test-case" data-status="{{.Status}}">
                    <div class="test-case-header" onclick="toggleTestCase(this)">
                        <div class="test-case-title">
                            <span class="chevron">▶</span>
                            <span class="badge {{if eq .Status "PASSED"}}passed{{else}}failed{{end}}">{{.Status}}</span>
                            <span>{{.Name}}</span>
                        </div>
                        <div class="test-case-meta">
                            <span>Duration: {{formatDuration .Duration}}</span>
                        </div>
                    </div>
                    <div class="test-case-body">
                        {{if .ErrorMsg}}
                        <div style="background-color: rgba(239, 68, 68, 0.1); border: 1px solid var(--accent-failure); padding: 1rem; border-radius: 0.375rem; margin-bottom: 1.5rem; color: #fecaca; font-family: monospace;">
                            <strong>Error:</strong> {{.ErrorMsg}}
                        </div>
                        {{end}}

                        <div class="timeline">
                            {{range .Steps}}
                            <div class="step-item">
                                <div class="step-dot {{if eq .Status "PASSED"}}passed{{else if eq .Status "FAILED"}}failed{{else}}info{{end}}"></div>
                                <div class="step-header">
                                    <div class="step-name">{{.Name}}</div>
                                    <div class="step-time">{{formatTime .Timestamp}}</div>
                                </div>
                                <div class="step-message">{{.Message}}</div>

                                {{if .Screenshot}}
                                <div class="screenshot-container" onclick="openModal('{{.Screenshot}}')">
                                    <img src="{{.Screenshot}}" alt="Screenshot evidence">
                                </div>
                                {{end}}

                                {{if or .APIRequest .APIResponse}}
                                <div class="api-dump">
                                    <div class="api-header" onclick="toggleAPIInspector(this, event)">
                                        <span>
                                            <span class="http-method {{.APIRequest.Method}}">{{.APIRequest.Method}}</span>
                                            <span style="margin-left: 0.5rem">{{.APIRequest.URL}}</span>
                                        </span>
                                        <span>Show Details ▼</span>
                                    </div>
                                    <div class="api-details" style="display: none;">
                                        {{if .APIRequest}}
                                        <div>
                                            <div class="api-section-title">Request Headers</div>
                                            <pre>{{jsonPretty .APIRequest.Headers}}</pre>
                                        </div>
                                        {{if .APIRequest.Body}}
                                        <div>
                                            <div class="api-section-title">Request Body</div>
                                            <pre>{{.APIRequest.Body}}</pre>
                                        </div>
                                        {{end}}
                                        {{end}}

                                        {{if .APIResponse}}
                                        <div>
                                            <div class="api-section-title">Response Status: {{.APIResponse.Status}}</div>
                                            <div class="api-section-title" style="margin-top: 0.5rem">Response Headers</div>
                                            <pre>{{jsonPretty .APIResponse.Headers}}</pre>
                                        </div>
                                        {{if .APIResponse.Body}}
                                        <div>
                                            <div class="api-section-title">Response Body</div>
                                            <pre>{{.APIResponse.Body}}</pre>
                                        </div>
                                        {{end}}
                                        {{end}}
                                    </div>
                                </div>
                                {{end}}
                            </div>
                            {{end}}
                        </div>
                    </div>
                </div>
                {{end}}
            </div>
        </section>
        {{end}}
    </div>

    <!-- Image Modal -->
    <div id="imageModal" class="modal" onclick="closeModal()">
        <img id="modalImg" src="" alt="Fullscreen Screenshot">
    </div>

    <script>
        function toggleTestCase(header) {
            header.classList.toggle('expanded');
            const chevron = header.querySelector('.chevron');
            if (header.classList.contains('expanded')) {
                chevron.textContent = '▼';
            } else {
                chevron.textContent = '▶';
            }
        }

        function toggleAPIInspector(header, event) {
            event.stopPropagation();
            const details = header.nextElementSibling;
            const textSpan = header.querySelector('span:last-child');
            if (details.style.display === 'none') {
                details.style.display = 'flex';
                textSpan.textContent = 'Hide Details ▲';
            } else {
                details.style.display = 'none';
                textSpan.textContent = 'Show Details ▼';
            }
        }

        function filterTests(status) {
            // Update active button styling
            const buttons = document.querySelectorAll('.btn-filter');
            buttons.forEach(btn => btn.classList.remove('active'));
            event.target.classList.add('active');

            // Filter test cases
            const testCases = document.querySelectorAll('.test-case');
            testCases.forEach(tc => {
                if (status === 'all') {
                    tc.style.display = '';
                } else if (tc.getAttribute('data-status') === status) {
                    tc.style.display = '';
                } else {
                    tc.style.display = 'none';
                }
            });

            // Filter suites (hide suite if it has no visible tests after filtering)
            const suites = document.querySelectorAll('.suite-section');
            suites.forEach(suite => {
                const visibleTests = suite.querySelectorAll('.test-case[style=""]');
                const allTests = suite.querySelectorAll('.test-case');
                
                let hasVisible = false;
                if (status === 'all') {
                    hasVisible = true;
                } else {
                    allTests.forEach(tc => {
                        if (tc.getAttribute('data-status') === status) {
                            hasVisible = true;
                        }
                    });
                }
                
                if (hasVisible) {
                    suite.style.display = '';
                } else {
                    suite.style.display = 'none';
                }
            });
        }

        function openModal(src) {
            document.getElementById('modalImg').src = src;
            document.getElementById('imageModal').style.display = 'flex';
        }

        function closeModal() {
            document.getElementById('imageModal').style.display = 'none';
        }
    </script>
</body>
</html>
`
