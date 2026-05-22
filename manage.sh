#!/usr/bin/env bash

# Antigravity Monorepo Management CLI Script
# Handles: install, build, run, stop, kill, status, troubleshoot, test

set -euo pipefail

# Terminal Color Codes
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Directories
WORKSPACE_DIR="/home/ubuntu/code/local/antigravity"
BACKEND_DIR="${WORKSPACE_DIR}/backend"
FRONTEND_DIR="${WORKSPACE_DIR}/frontend"
E2E_DIR="${WORKSPACE_DIR}/e2etest"

# PID files
BACKEND_PID_FILE="${WORKSPACE_DIR}/.backend.pid"
FRONTEND_PID_FILE="${WORKSPACE_DIR}/.frontend.pid"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_help() {
    echo -e "${CYAN}================================================================${NC}"
    echo -e "         ${GREEN}Antigravity Monorepo Administration Tool${NC}"
    echo -e "${CYAN}================================================================${NC}"
    echo -e "Usage: ./manage.sh <command> [arguments]"
    echo
    echo -e "Commands:"
    echo -e "  ${GREEN}install${NC}       Installs dependencies for backend, frontend, and E2E tests"
    echo -e "  ${GREEN}build${NC}         Generates swagger docs and compiles all components"
    echo -e "  ${GREEN}run [target]${NC}  Runs services in background. Target: backend, frontend, all (default)"
    echo -e "  ${GREEN}stop${NC}          Gracefully stops background servers"
    echo -e "  ${GREEN}kill${NC}          Force kills processes running on port 8080 and 5173"
    echo -e "  ${GREEN}status${NC}        Displays PIDs, ports, and activity status of components"
    echo -e "  ${GREEN}troubleshoot${NC}  Diagnoses environment tools, configs, and system dependencies"
    echo -e "  ${GREEN}test [target]${NC} Runs test suites. Target: backend, frontend, e2e, all (default)"
    echo -e "  ${GREEN}swagger [env]${NC} Pre-generates Swagger docs customized for env: local, dev, tst, prd"
    echo -e "${CYAN}================================================================${NC}"
}

do_install() {
    log_info "Installing Monorepo Dependencies..."
    
    # Backend
    log_info "Tidying Go backend dependencies..."
    cd "${BACKEND_DIR}"
    go mod tidy
    
    # Frontend
    log_info "Installing React frontend dependencies (npm)..."
    cd "${FRONTEND_DIR}"
    npm install
    
    # E2E
    log_info "Tidying E2E tests Go dependencies..."
    cd "${E2E_DIR}"
    go mod tidy
    
    log_success "All dependencies installed successfully!"
}

do_swagger() {
    local env="${1:-local}"
    log_info "Generating Swagger docs for environment: ${env}..."
    
    cd "${BACKEND_DIR}"
    # Run Swag CLI
    if command -v /home/ubuntu/go/bin/swag &> /dev/null; then
        /home/ubuntu/go/bin/swag init -g cmd/server/main.go -o docs
    else
        log_warn "swag CLI not found in standard path. Attempting to install..."
        go install github.com/swaggo/swag/cmd/swag@latest
        /home/ubuntu/go/bin/swag init -g cmd/server/main.go -o docs
    fi
    
    # Dynamic swagger per-environment logic. The backend/internal/router/router.go
    # takes care of modifying host/title/schemes dynamically based on ActiveConfig at runtime!
    log_success "Swagger docs initialized successfully! Configured to dynamically load for: ${env}"
}

do_build() {
    log_info "Building components..."
    
    # Swagger docs first
    do_swagger "local"
    
    # Backend binary
    log_info "Compiling Go backend server..."
    cd "${BACKEND_DIR}"
    mkdir -p bin
    go build -o bin/server cmd/server/main.go
    
    # Frontend build
    log_info "Compiling React production bundle..."
    cd "${FRONTEND_DIR}"
    npm run build
    
    log_success "Build completed successfully!"
}

do_run() {
    local target="${1:-all}"
    mkdir -p "${WORKSPACE_DIR}/logs"
    
    # Ensure port is clean
    if [ "$target" = "backend" ] || [ "$target" = "all" ]; then
        if lsof -i :8080 >/dev/null 2>&1; then
            log_warn "Port 8080 is already in use. Attempting to stop..."
            do_kill
        fi
    fi
    
    if [ "$target" = "frontend" ] || [ "$target" = "all" ]; then
        if lsof -i :5173 >/dev/null 2>&1; then
            log_warn "Port 5173 is already in use. Attempting to stop..."
            do_kill
        fi
    fi

    # Run Backend
    if [ "$target" = "backend" ] || [ "$target" = "all" ]; then
        log_info "Starting Go backend in background..."
        cd "${BACKEND_DIR}"
        export APP_ENV=local
        # Run binary in background
        ./bin/server > "${WORKSPACE_DIR}/logs/backend.log" 2>&1 &
        local backend_pid=$!
        echo $backend_pid > "${BACKEND_PID_FILE}"
        log_success "Backend started with PID ${backend_pid} (Logs: logs/backend.log)"
    fi

    # Run Frontend
    if [ "$target" = "frontend" ] || [ "$target" = "all" ]; then
        log_info "Starting React frontend in background..."
        cd "${FRONTEND_DIR}"
        # Start Vite dev server non-interactively
        npx vite --host 127.0.0.1 --port 5173 > "${WORKSPACE_DIR}/logs/frontend.log" 2>&1 &
        local frontend_pid=$!
        echo $frontend_pid > "${FRONTEND_PID_FILE}"
        log_success "Frontend started with PID ${frontend_pid} (Logs: logs/frontend.log)"
    fi
    
    # Wait for startup confirmation
    log_info "Verifying services startup..."
    sleep 3
    do_status
}

do_stop() {
    log_info "Stopping background services..."
    
    # Stop Frontend
    if [ -f "${FRONTEND_PID_FILE}" ]; then
        local pid=$(cat "${FRONTEND_PID_FILE}")
        log_info "Stopping frontend process (PID ${pid})..."
        kill -15 "$pid" 2>/dev/null || true
        rm -f "${FRONTEND_PID_FILE}"
        log_success "Frontend stopped."
    else
        log_info "No frontend PID file found."
    fi

    # Stop Backend
    if [ -f "${BACKEND_PID_FILE}" ]; then
        local pid=$(cat "${BACKEND_PID_FILE}")
        log_info "Stopping backend process (PID ${pid})..."
        kill -15 "$pid" 2>/dev/null || true
        rm -f "${BACKEND_PID_FILE}"
        log_success "Backend stopped."
    else
        log_info "No backend PID file found."
    fi
}

do_kill() {
    log_warn "Force killing processes on active monorepo ports..."
    
    # Kill backend
    local backend_pids=$(lsof -t -i :8080 || true)
    if [ -n "$backend_pids" ]; then
        log_warn "Force killing backend listeners on port 8080: ${backend_pids}"
        kill -9 $backend_pids 2>/dev/null || true
    fi
    rm -f "${BACKEND_PID_FILE}"
    
    # Kill frontend
    local frontend_pids=$(lsof -t -i :5173 || true)
    if [ -n "$frontend_pids" ]; then
        log_warn "Force killing frontend listeners on port 5173: ${frontend_pids}"
        kill -9 $frontend_pids 2>/dev/null || true
    fi
    rm -f "${FRONTEND_PID_FILE}"
    
    # Kill orphaned Chromedriver/Selenium
    local selenium_pids=$(lsof -t -i :8082 || true)
    if [ -n "$selenium_pids" ]; then
        log_info "Cleaning up leftover E2E Chromedriver processes..."
        kill -9 $selenium_pids 2>/dev/null || true
    fi
    
    log_success "Process cleanup completed."
}

do_status() {
    echo -e "${CYAN}----------------------------------------------------------------${NC}"
    echo -e "                 ${GREEN}Service Activity Status Table${NC}"
    echo -e "${CYAN}----------------------------------------------------------------${NC}"
    printf "%-15s %-10s %-10s %-20s\n" "SERVICE" "PID" "PORT" "STATUS"
    echo -e "${CYAN}----------------------------------------------------------------${NC}"
    
    # Check Backend
    local backend_status="INACTIVE"
    local backend_pid="-"
    if [ -f "${BACKEND_PID_FILE}" ]; then
        backend_pid=$(cat "${BACKEND_PID_FILE}")
        if kill -0 "$backend_pid" 2>/dev/null; then
            backend_status="${GREEN}ACTIVE (PID file)${NC}"
        else
            backend_status="${RED}CRASHED (Stale PID)${NC}"
        fi
    elif lsof -i :8080 >/dev/null 2>&1; then
        backend_pid=$(lsof -t -i :8080 | head -n1)
        backend_status="${GREEN}ACTIVE (Listening)${NC}"
    fi
    printf "%-15s %-10s %-10s %b\n" "Backend (Go)" "$backend_pid" "8080" "$backend_status"
    
    # Check Frontend
    local frontend_status="INACTIVE"
    local frontend_pid="-"
    if [ -f "${FRONTEND_PID_FILE}" ]; then
        frontend_pid=$(cat "${FRONTEND_PID_FILE}")
        if kill -0 "$frontend_pid" 2>/dev/null; then
            frontend_status="${GREEN}ACTIVE (PID file)${NC}"
        else
            frontend_status="${RED}CRASHED (Stale PID)${NC}"
        fi
    elif lsof -i :5173 >/dev/null 2>&1; then
        frontend_pid=$(lsof -t -i :5173 | head -n1)
        frontend_status="${GREEN}ACTIVE (Listening)${NC}"
    fi
    printf "%-15s %-10s %-10s %b\n" "Frontend (React)" "$frontend_pid" "5173" "$frontend_status"
    echo -e "${CYAN}----------------------------------------------------------------${NC}"
}

do_troubleshoot() {
    log_info "Running System Diagnostic Self-Checks..."
    echo
    
    # 1. System tool checks
    echo -e "${CYAN}[Step 1] Verifying System Toolchains:${NC}"
    
    if command -v go &> /dev/null; then
        echo -e "  Go:        ${GREEN}OK${NC} ($(go version))"
    else
        echo -e "  Go:        ${RED}MISSING${NC}"
    fi
    
    if command -v node &> /dev/null; then
        echo -e "  Node:      ${GREEN}OK${NC} ($(node -v))"
    else
        echo -e "  Node:      ${RED}MISSING${NC}"
    fi
    
    if command -v npm &> /dev/null; then
        echo -e "  NPM:       ${GREEN}OK${NC} ($(npm -v))"
    else
        echo -e "  NPM:       ${RED}MISSING${NC}"
    fi
    echo
    
    # 2. Browser dependencies check
    echo -e "${CYAN}[Step 2] Verifying Browser Dependencies (E2E):${NC}"
    
    if [ -f "/usr/bin/chromedriver" ]; then
        echo -e "  Chromedriver: ${GREEN}OK${NC} (/usr/bin/chromedriver)"
    else
        echo -e "  Chromedriver: ${RED}MISSING (/usr/bin/chromedriver)${NC}"
    fi
    
    if [ -f "/usr/bin/chromium" ]; then
        echo -e "  Chromium:     ${GREEN}OK${NC} (/usr/bin/chromium)"
    else
        echo -e "  Chromium:     ${RED}MISSING (/usr/bin/chromium)${NC}"
    fi
    echo
    
    # 3. Monorepo directory checks
    echo -e "${CYAN}[Step 3] Checking Workspace Directories:${NC}"
    for dir in "${BACKEND_DIR}" "${FRONTEND_DIR}" "${E2E_DIR}"; do
        if [ -d "$dir" ]; then
            echo -e "  $(basename "$dir"): ${GREEN}OK${NC}"
        else
            echo -e "  $(basename "$dir"): ${RED}MISSING${NC}"
        fi
    done
    echo
    
    # 4. Port Availability checks
    echo -e "${CYAN}[Step 4] Port Conflicts Auditing:${NC}"
    for port in 8080 5173 8082; do
        if lsof -i :$port >/dev/null 2>&1; then
            local service_on_port=$(lsof -i :$port -sTCP:LISTEN -Fp | head -n1 | tr -d 'p' || echo "Unknown")
            echo -e "  Port ${port}: ${RED}IN USE${NC} (by PID ${service_on_port})"
        else
            echo -e "  Port ${port}: ${GREEN}AVAILABLE${NC}"
        fi
    done
    echo
    
    log_success "Diagnostic self-check complete."
}

do_test() {
    local target="${1:-all}"
    
    if [ "$target" = "backend" ]; then
        log_info "Running Go backend unit tests..."
        cd "${BACKEND_DIR}"
        go test -v ./...
        
    elif [ "$target" = "frontend" ]; then
        log_info "Running frontend checks..."
        cd "${FRONTEND_DIR}"
        # Vite projects don't have unit tests by default, but we check linting/building
        npm run build
        
    elif [ "$target" = "e2e" ]; then
        log_info "Running Automated Selenium & API E2E tests..."
        
        # Ensure server components are active
        local spin_up=false
        if ! lsof -i :8080 >/dev/null 2>&1 || ! lsof -i :5173 >/dev/null 2>&1; then
            log_info "Monorepo services are inactive. Launching test cluster..."
            spin_up=true
            do_run "all"
            sleep 4
        fi
        
        # Run tests
        log_info "Executing E2E test suite..."
        cd "${E2E_DIR}"
        local test_status=0
        export APP_ENV=tst
        export BACKEND_URL="http://localhost:8080"
        export FRONTEND_URL="http://localhost:5173"
        go test -v ./... || test_status=$?
        
        # Stop servers if we spun them up
        if [ "$spin_up" = true ]; then
            log_info "Tearing down test cluster..."
            do_stop
        fi
        
        if [ $test_status -eq 0 ]; then
            log_success "All E2E UI and API tests PASSED!"
        else
            log_error "E2E tests FAILED (exit code: ${test_status})"
            exit $test_status
        fi
        
    elif [ "$target" = "all" ]; then
        log_info "Running full test suites..."
        cd "${BACKEND_DIR}"
        go test -v ./...
        
        # Run E2E
        do_test "e2e"
    else
        log_error "Unknown test target: ${target}"
        exit 1
    fi
}

# Main routing logic
if [ $# -lt 1 ]; then
    print_help
    exit 0
fi

CMD="$1"
shift

case "$CMD" in
    install)
        do_install
        ;;
    build)
        do_build
        ;;
    run)
        do_run "${1:-all}"
        ;;
    stop)
        do_stop
        ;;
    kill)
        do_kill
        ;;
    status)
        do_status
        ;;
    troubleshoot)
        do_troubleshoot
        ;;
    test)
        do_test "${1:-all}"
        ;;
    swagger)
        do_swagger "${1:-local}"
        ;;
    help|--help|-h)
        print_help
        ;;
    *)
        log_error "Unknown command: ${CMD}"
        print_help
        exit 1
        ;;
esac
