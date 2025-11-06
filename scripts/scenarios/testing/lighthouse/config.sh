#!/usr/bin/env bash
# Default configuration and utilities for Lighthouse testing
# Provides fallback thresholds and helper functions

# Default thresholds (can be overridden per-page in .lighthouse/config.json)
LIGHTHOUSE_DEFAULT_PERFORMANCE_ERROR=0.70
LIGHTHOUSE_DEFAULT_PERFORMANCE_WARN=0.80
LIGHTHOUSE_DEFAULT_ACCESSIBILITY_ERROR=0.85
LIGHTHOUSE_DEFAULT_ACCESSIBILITY_WARN=0.90
LIGHTHOUSE_DEFAULT_BEST_PRACTICES_ERROR=0.80
LIGHTHOUSE_DEFAULT_BEST_PRACTICES_WARN=0.90
LIGHTHOUSE_DEFAULT_SEO_ERROR=0.75
LIGHTHOUSE_DEFAULT_SEO_WARN=0.85

# Default Chrome flags
LIGHTHOUSE_DEFAULT_CHROME_FLAGS=(
  "--headless"
  "--no-sandbox"
  "--disable-gpu"
  "--disable-dev-shm-usage"
  "--disable-extensions"
)

# Default Lighthouse categories
LIGHTHOUSE_DEFAULT_CATEGORIES=(
  "performance"
  "accessibility"
  "best-practices"
  "seo"
)

# Default viewport dimensions
LIGHTHOUSE_DESKTOP_WIDTH=1440
LIGHTHOUSE_DESKTOP_HEIGHT=900
LIGHTHOUSE_MOBILE_WIDTH=375
LIGHTHOUSE_MOBILE_HEIGHT=667

# Create a minimal Lighthouse config template
lighthouse::create_template_config() {
  local output_path="${1}"
  local scenario_name="${2:-my-scenario}"

  cat > "$output_path" <<'EOF'
{
  "_metadata": {
    "description": "Lighthouse testing configuration",
    "last_updated": "TIMESTAMP"
  },
  "enabled": true,
  "pages": [
    {
      "id": "home",
      "path": "/",
      "label": "Home Page",
      "viewport": "desktop",
      "thresholds": {
        "performance": { "error": 0.70, "warn": 0.80 },
        "accessibility": { "error": 0.85, "warn": 0.90 },
        "best-practices": { "error": 0.80, "warn": 0.90 },
        "seo": { "error": 0.75, "warn": 0.85 }
      },
      "requirements": []
    }
  ],
  "global_options": {
    "lighthouse": {
      "extends": "lighthouse:default",
      "settings": {
        "onlyCategories": ["performance", "accessibility", "best-practices", "seo"],
        "throttlingMethod": "simulate",
        "formFactor": "desktop"
      }
    },
    "chrome_flags": ["--headless", "--no-sandbox", "--disable-gpu"],
    "timeout_ms": 60000,
    "retries": 2
  },
  "reporting": {
    "output_dir": "test/artifacts/lighthouse",
    "formats": ["json", "html"],
    "keep_reports": 10,
    "fail_on_error": true,
    "fail_on_warn": false
  }
}
EOF

  # Replace timestamp
  if command -v sed >/dev/null 2>&1; then
    sed -i "s/TIMESTAMP/$(date -u +%Y-%m-%dT%H:%M:%SZ)/" "$output_path"
  fi

  echo "Created Lighthouse config template: $output_path"
  echo "Edit this file to add more pages and adjust thresholds"
}

# Initialize Lighthouse testing for a scenario
lighthouse::init_scenario() {
  local scenario_dir="${1}"

  if [ -z "$scenario_dir" ] || [ ! -d "$scenario_dir" ]; then
    echo "Error: Invalid scenario directory: $scenario_dir" >&2
    return 1
  fi

  local scenario_name=$(basename "$scenario_dir")
  local lighthouse_dir="${scenario_dir}/.lighthouse"
  local config_file="${lighthouse_dir}/config.json"

  # Create .lighthouse directory
  if [ ! -d "$lighthouse_dir" ]; then
    mkdir -p "$lighthouse_dir"
    echo "Created directory: $lighthouse_dir"
  fi

  # Create config template if it doesn't exist
  if [ ! -f "$config_file" ]; then
    lighthouse::create_template_config "$config_file" "$scenario_name"
  else
    echo "Config already exists: $config_file"
  fi

  # Create artifacts directory
  local artifacts_dir="${scenario_dir}/test/artifacts/lighthouse"
  if [ ! -d "$artifacts_dir" ]; then
    mkdir -p "$artifacts_dir"
    echo "Created artifacts directory: $artifacts_dir"
  fi

  # Add to .gitignore if it exists
  local gitignore="${scenario_dir}/.gitignore"
  if [ -f "$gitignore" ]; then
    if ! grep -q "test/artifacts/lighthouse" "$gitignore"; then
      echo "" >> "$gitignore"
      echo "# Lighthouse test artifacts" >> "$gitignore"
      echo "test/artifacts/lighthouse/*.html" >> "$gitignore"
      echo "test/artifacts/lighthouse/*.json" >> "$gitignore"
      echo "Added Lighthouse artifacts to .gitignore"
    fi
  fi

  echo ""
  echo "✅ Lighthouse testing initialized for $scenario_name"
  echo ""
  echo "Next steps:"
  echo "  1. Edit $config_file to configure pages and thresholds"
  echo "  2. Add Lighthouse requirements to requirements/*.json"
  echo "  3. Run tests: ./test/run-tests.sh --phases performance"
  echo ""
}

# Validate that required dependencies are installed
lighthouse::check_dependencies() {
  local errors=0

  if ! command -v node >/dev/null 2>&1; then
    echo "❌ Node.js not found (required for Lighthouse)" >&2
    echo "   Install Node.js 16+ from https://nodejs.org" >&2
    errors=$((errors + 1))
  else
    local node_version
    node_version=$(node --version | sed 's/v//' | cut -d. -f1)
    if [ "$node_version" -lt 16 ]; then
      echo "❌ Node.js version too old: $(node --version)" >&2
      echo "   Lighthouse requires Node.js 16+" >&2
      errors=$((errors + 1))
    fi
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "⚠️  jq not found (recommended for config parsing)" >&2
    echo "   Install: apt-get install jq (or brew install jq)" >&2
  fi

  # Check if Lighthouse is installed
  local lighthouse_dir
  lighthouse_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  if [ ! -d "${lighthouse_dir}/node_modules" ]; then
    echo "⚠️  Lighthouse dependencies not installed" >&2
    echo "   Run: cd $lighthouse_dir && npm install" >&2
  fi

  return $errors
}

# Get score color/emoji based on thresholds
lighthouse::get_score_emoji() {
  local score="${1}"
  local error_threshold="${2:-0.70}"
  local warn_threshold="${3:-0.80}"

  # Convert to integer for comparison (multiply by 100)
  local score_int=$(echo "$score * 100" | bc | cut -d. -f1)
  local error_int=$(echo "$error_threshold * 100" | bc | cut -d. -f1)
  local warn_int=$(echo "$warn_threshold * 100" | bc | cut -d. -f1)

  if [ "$score_int" -lt "$error_int" ]; then
    echo "❌"
  elif [ "$score_int" -lt "$warn_int" ]; then
    echo "⚠️ "
  else
    echo "✅"
  fi
}

# Export functions if being sourced
if [ "${BASH_SOURCE[0]}" != "${0}" ]; then
  export -f lighthouse::create_template_config 2>/dev/null || true
  export -f lighthouse::init_scenario 2>/dev/null || true
  export -f lighthouse::check_dependencies 2>/dev/null || true
  export -f lighthouse::get_score_emoji 2>/dev/null || true
fi
