# 🚀 Antigravity Fullstack Monorepo Template

Welcome to the **Antigravity Fullstack Monorepo Template**! This repository is a premium starting point that includes a Go REST API backend, a modern React (Vite) frontend, and a highly structured, professional Selenium & API End-to-End (E2E) testing framework with automatic interactive HTML reporting archived inside distinct, time-stamped folders.

---

## 📂 Monorepo Architecture & Directory Tour

This workspace is organized as a unified monorepo:

```
├── backend/             # Go REST API Server
│   ├── cmd/server/      # Main entrypoint
│   ├── internal/        # Core packages (auth, health check, router, middleware)
│   └── docs/            # Auto-generated Swagger documentation files
├── frontend/            # React + Vite Client Web Application
│   ├── src/             # Core UI components, styles, views, assets
│   └── public/          # Static files and entry points
├── e2etest/             # Restructured E2E Test Suite (Web UI & API tests)
│   ├── config.go        # JSON parser and sample config auto-generator
│   ├── e2e-test.json    # External test fixtures JSON file (generated dynamically)
│   ├── base_suite.go    # Reusable Testify suite bases (BaseAPISuite, BaseWebSuite)
│   ├── reporter.go      # Custom HTML interactive reporting engine
│   ├── api_suite_test.go# Reusable API test suite (endpoints validation)
│   ├── web_suite_test.go# Headless/Graphical browser workflow automation
│   └── reports/         # Dynamic time-stamped E2E reports directory
│       └── run_YYYY-MM-DD_HH-MM-SS/   # Example E2E run archive
│           ├── report.html            # Run-specific interactive HTML dashboard
│           └── report_evidences/      # Visual evidence screenshots for this run
└── manage.sh            # Monorepo Management CLI Script (portable & dynamic)
```

---

## ⚡ Quick Start Guide (Kick Start)

Follow these three commands to get the entire project up and running:

### 1. Install All Dependencies
Installs required packages for the Go backend, React frontend, and E2E test suites in one go:
```bash
./manage.sh install
```

### 2. Build the Monorepo Components
Generates Swagger documentation and compiles the backend REST server and frontend client:
```bash
./manage.sh build
```

### 3. Spin Up the Development Services
Launches the Go API Server on port `8080` and the React Vite Server on port `5173`:
```bash
./manage.sh run
```
*To verify service health, run `./manage.sh status`. To stop services gracefully, run `./manage.sh stop`.*

---

## 🛡️ Professional E2E Testing Suite (`e2etest`)

The E2E tests are organized as structured, object-oriented test suites using `github.com/stretchr/testify/suite` and log step-by-step evidences (screenshots and network request/response dumps) to an interactive HTML dashboard.

### Run All End-to-End Tests
To automatically spin up the test cluster, execute all API and browser tests, write reports, and tear down the cluster, run:
```bash
./manage.sh test e2e
```
*This command uses `-count=1` under the hood to completely bypass Go's test caching and force fresh execution.*

### 🛠️ Configuration & Live View Toggles (`e2e-test.json`)
The tests load fixtures dynamically from `e2etest/e2e-test.json`. If this file is missing, it is **automatically generated** with standard defaults on the first test run.

Open [e2etest/e2e-test.json](file:///home/ubuntu/code/github/raviautopilot/fullstack-template/e2etest/e2e-test.json) to customize test behavior:

```json
{
  "frontend_url": "http://localhost:5173",
  "backend_url": "http://localhost:8080",
  "chromedriver_path": "/usr/bin/chromedriver",
  "chromium_path": "/usr/bin/chromium",
  "chromedriver_port": 8082,
  "headless": true,
  "enable_evidence": true,
  "evidence_dir": "report_evidences",
  "report_path": "report.html"
}
```

#### 📺 Visual Mode (How to watch the browser running)
By default, the Selenium tests run in `headless` mode (ideal for CI/CD environments). If you want to disable headless mode and **watch the physical Chrome browser launch, perform clicks, and automate the test flow on screen**, simply update the toggle:
- Change `"headless": true` to `"headless": false`.

#### 📸 Evidence & Report Customization
- **`enable_evidence`**: Set to `false` to completely disable capturing visual screenshots and HTTP request/response payloads, producing a lightweight, text-only timeline.
- **`evidence_dir`**: The folder where captured screenshots are saved.
- **`report_path`**: The file name of the generated interactive HTML report (defaults to `report.html`).

---

## 🕒 Run Archiving & Comparability (`reports/`)

Every time E2E tests run via `./manage.sh test e2e`, a **distinct, time-stamped directory** is created under `e2etest/reports/`:
```
e2etest/reports/run_YYYY-MM-DD_HH-MM-SS/
```

This directory isolates that specific run's assets completely:
- The dashboard report page `report.html` is written directly inside.
- Visual screenshot assets are isolated under its local `report_evidences/` folder.

This isolated layout is fully self-contained using relative references, meaning the entire directory can be packed, shared, or compared side-by-side in separate browser tabs for regression review.

---

## 🛠️ Management CLI Cheat Sheet (`manage.sh`)

| Command | Action |
|:---|:---|
| `./manage.sh install` | Installs dependencies for backend, frontend, and E2E modules. |
| `./manage.sh build` | Re-generates Swagger API documentation and compiles all applications. |
| `./manage.sh run` | Launches the monorepo dev stack in the background (Go REST & Vite React). |
| `./manage.sh stop` | Gracefully terminates background servers and cleans up PID locks. |
| `./manage.sh status` | Displays PID, port, and health check activity for active background servers. |
| `./manage.sh test backend` | Runs the Go backend package unit tests. |
| `./manage.sh test frontend` | Builds the React frontend application bundle to verify compile check. |
| `./manage.sh test e2e` | Runs E2E browser and API test suites, writing evidence to a time-stamped folder. |
| `./manage.sh test all` | Runs full backend unit, frontend compiler check, and E2E test pipelines. |
| `./manage.sh troubleshoot` | Runs an automated network diagnostic check on monorepo dependencies. |

---

## 📊 Viewing the Test Report Dashboard

Once the E2E tests execute, locate and open the compiled time-stamped report:
```
e2etest/reports/run_YYYY-MM-DD_HH-MM-SS/report.html
```

It lists pass/fail metrics, execution timings, step-by-step logs, collapsible HTTP network blocks, and clickable high-resolution visual thumbnails.
