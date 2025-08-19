#!/usr/bin/env bash
# K6 Installation Functions

# Install K6
k6::install::execute() {
    log::info "Installing K6 load testing tool..."
    
    # Initialize directories
    k6::core::init
    
    # Detect installation method
    if command -v snap >/dev/null 2>&1; then
        k6::install::snap
    elif command -v apt-get >/dev/null 2>&1; then
        k6::install::apt
    else
        k6::install::binary
    fi
    
    # Create sample test scripts
    k6::install::create_samples
    
    # Register with Vrooli
    k6::install::register
    
    log::success "K6 installed successfully"
}

# Install via snap
k6::install::snap() {
    log::info "Installing K6 via snap..."
    sudo snap install k6
}

# Install via apt
k6::install::apt() {
    log::info "Installing K6 via apt..."
    
    # Add Grafana GPG key
    sudo gpg -k
    sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
        --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69 2>/dev/null || \
    wget -q -O - https://dl.k6.io/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/k6-archive-keyring.gpg
    
    # Add K6 repository
    echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
        sudo tee /etc/apt/sources.list.d/k6.list
    
    # Update and install
    sudo apt-get update
    sudo apt-get install -y k6
}

# Install via binary download
k6::install::binary() {
    log::info "Installing K6 via binary download..."
    
    local version="v0.49.0"  # Latest stable as of 2024
    local arch=$(uname -m)
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    
    # Map architecture
    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) log::error "Unsupported architecture: $arch"; return 1 ;;
    esac
    
    # Download binary
    local url="https://github.com/grafana/k6/releases/download/${version}/k6-${version}-${os}-${arch}.tar.gz"
    local tmp_dir=$(mktemp -d)
    
    log::info "Downloading K6 from $url..."
    wget -q -O "${tmp_dir}/k6.tar.gz" "$url"
    
    # Extract and install
    tar -xzf "${tmp_dir}/k6.tar.gz" -C "$tmp_dir"
    sudo mv "${tmp_dir}/k6-${version}-${os}-${arch}/k6" /usr/local/bin/
    sudo chmod +x /usr/local/bin/k6
    
    # Cleanup
    rm -rf "$tmp_dir"
}

# Create sample test scripts
k6::install::create_samples() {
    log::info "Creating sample test scripts..."
    
    # Basic HTTP test
    cat > "$K6_SCRIPTS_DIR/basic-http.js" << 'EOF'
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,
  duration: '30s',
};

export default function () {
  const res = http.get('https://test.k6.io');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  sleep(1);
}
EOF

    # API load test
    cat > "$K6_SCRIPTS_DIR/api-load.js" << 'EOF'
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 20 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.1'],
  },
};

export default function () {
  const res = http.get('https://httpbin.org/get');
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
EOF

    # Stress test
    cat > "$K6_SCRIPTS_DIR/stress-test.js" << 'EOF'
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '2m', target: 100 },
    { duration: '5m', target: 100 },
    { duration: '2m', target: 200 },
    { duration: '5m', target: 200 },
    { duration: '2m', target: 300 },
    { duration: '5m', target: 300 },
    { duration: '2m', target: 0 },
  ],
};

export default function () {
  const res = http.get('https://test.k6.io');
  check(res, { 'status is 200': (r) => r.status === 200 });
  sleep(1);
}
EOF
    
    log::success "Sample test scripts created in $K6_SCRIPTS_DIR"
}

# Register with Vrooli
k6::install::register() {
    # Create CLI symlink if needed
    if [[ ! -L "/usr/local/bin/resource-k6" ]]; then
        sudo ln -sf "${K6_CLI_DIR}/resource-k6" /usr/local/bin/resource-k6
    fi
    
    # Auto-register if vrooli command exists
    if command -v vrooli >/dev/null 2>&1; then
        log::info "Registering K6 with Vrooli..."
        # Registration happens automatically via CLI discovery
    fi
}

# Validate installation
k6::install::validate() {
    log::info "Validating K6 installation..."
    
    local errors=0
    
    # Check K6 binary
    if ! command -v k6 >/dev/null 2>&1; then
        log::error "K6 binary not found"
        ((errors++))
    else
        log::success "K6 binary found: $(which k6)"
    fi
    
    # Check version
    if command -v k6 >/dev/null 2>&1; then
        local version=$(k6 version 2>/dev/null)
        log::success "K6 version: $version"
    fi
    
    # Check directories
    for dir in "$K6_DATA_DIR" "$K6_SCRIPTS_DIR" "$K6_RESULTS_DIR"; do
        if [[ -d "$dir" ]]; then
            log::success "Directory exists: $dir"
        else
            log::error "Directory missing: $dir"
            ((errors++))
        fi
    done
    
    # Check sample scripts
    local script_count=$(find "$K6_SCRIPTS_DIR" -name "*.js" 2>/dev/null | wc -l)
    if [[ $script_count -gt 0 ]]; then
        log::success "Found $script_count test scripts"
    else
        log::warn "No test scripts found"
    fi
    
    if [[ $errors -eq 0 ]]; then
        log::success "K6 validation passed"
        return 0
    else
        log::error "K6 validation failed with $errors errors"
        return 1
    fi
}

# Uninstall K6
k6::install::uninstall() {
    if [[ "$1" != "--force" ]]; then
        log::error "Uninstall requires --force flag"
        return 1
    fi
    
    log::warn "Uninstalling K6..."
    
    # Stop any running tests
    pkill -f k6 2>/dev/null || true
    
    # Remove binary
    if command -v k6 >/dev/null 2>&1; then
        if command -v snap >/dev/null 2>&1 && snap list k6 2>/dev/null; then
            sudo snap remove k6
        elif [[ -f /usr/local/bin/k6 ]]; then
            sudo rm -f /usr/local/bin/k6
        elif command -v apt-get >/dev/null 2>&1; then
            sudo apt-get remove -y k6
        fi
    fi
    
    # Remove CLI symlink
    sudo rm -f /usr/local/bin/resource-k6
    
    # Optionally remove data (prompt user)
    read -p "Remove K6 data directory? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$K6_DATA_DIR"
        log::info "K6 data removed"
    fi
    
    log::success "K6 uninstalled"
}