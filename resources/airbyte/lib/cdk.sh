#!/bin/bash
# Airbyte Connector Development Kit (CDK) Support

set -euo pipefail

# Resource metadata
RESOURCE_NAME="airbyte"
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${RESOURCE_DIR}/data"
CDK_DIR="${DATA_DIR}/cdk"

# Logging functions
log_info() {
    echo "[INFO] $*" >&2
}

log_error() {
    echo "[ERROR] $*" >&2
}

# Initialize CDK environment
cdk_init() {
    local name="${1:-custom-connector}"
    local type="${2:-source}"  # source or destination
    
    log_info "Initializing CDK environment for ${type} connector: ${name}..."
    
    # Create CDK directory if it doesn't exist
    mkdir -p "${CDK_DIR}"
    
    # Create connector directory
    local connector_dir="${CDK_DIR}/${type}-${name}"
    if [[ -d "$connector_dir" ]]; then
        log_error "Connector directory already exists: $connector_dir"
        return 1
    fi
    
    mkdir -p "$connector_dir"
    
    # Create basic Python CDK structure
    cat > "${connector_dir}/setup.py" <<'EOF'
from setuptools import setup, find_packages

setup(
    name="airbyte-CONNECTOR_NAME",
    description="CONNECTOR_TYPE connector for Airbyte",
    author="Airbyte",
    author_email="",
    packages=find_packages(),
    install_requires=[
        "airbyte-cdk",
        "requests",
    ],
    package_data={"": ["*.json", "*.yaml", "schemas/*.json", "schemas/shared/*.json"]},
    extras_require={
        "tests": [
            "pytest~=6.2",
            "pytest-mock~=3.6.1",
            "source-acceptance-test",
        ]
    },
)
EOF
    
    # Update setup.py with actual names
    sed -i "s/CONNECTOR_NAME/${name}/g" "${connector_dir}/setup.py"
    sed -i "s/CONNECTOR_TYPE/${type^}/g" "${connector_dir}/setup.py"
    
    # Create main connector module
    mkdir -p "${connector_dir}/${type}_${name//-/_}"
    
    # Create spec.yaml
    cat > "${connector_dir}/${type}_${name//-/_}/spec.yaml" <<EOF
documentationUrl: https://docs.airbyte.com/integrations/${type}s/${name}
connectionSpecification:
  \$schema: http://json-schema.org/draft-07/schema#
  title: ${type^} ${name} Spec
  type: object
  required:
    - api_key
  properties:
    api_key:
      type: string
      title: API Key
      description: API key for authentication
      airbyte_secret: true
EOF
    
    # Create __init__.py
    cat > "${connector_dir}/${type}_${name//-/_}/__init__.py" <<EOF
from .${type} import ${type^}${name//[-_]/''}
__all__ = ["${type^}${name//[-_]/''}"]
EOF
    
    # Create source or destination implementation
    if [[ "$type" == "source" ]]; then
        cat > "${connector_dir}/${type}_${name//-/_}/${type}.py" <<'EOF'
from typing import Any, Iterable, List, Mapping, MutableMapping, Optional, Tuple
from airbyte_cdk import AirbyteLogger
from airbyte_cdk.sources import AbstractSource
from airbyte_cdk.sources.streams import Stream
from airbyte_cdk.sources.streams.http import HttpStream
import requests

class CustomStream(HttpStream):
    url_base = "https://api.example.com/v1/"
    
    def path(self, **kwargs) -> str:
        return "data"
    
    def parse_response(self, response: requests.Response, **kwargs) -> Iterable[Mapping]:
        return response.json()

class SourceCUSTOM_NAME(AbstractSource):
    def check_connection(self, logger, config) -> Tuple[bool, any]:
        try:
            # Add connection check logic here
            return True, None
        except Exception as e:
            return False, str(e)
    
    def streams(self, config: Mapping[str, Any]) -> List[Stream]:
        return [CustomStream(authenticator=None)]
EOF
        sed -i "s/CUSTOM_NAME/${name//[-_]/''}/g" "${connector_dir}/${type}_${name//-/_}/${type}.py"
        
    else  # destination
        cat > "${connector_dir}/${type}_${name//-/_}/${type}.py" <<'EOF'
from typing import Any, Iterable, Mapping
from airbyte_cdk import AirbyteLogger
from airbyte_cdk.destinations import Destination

class DestinationCUSTOM_NAME(Destination):
    def write(
        self, config: Mapping[str, Any], configured_catalog, input_messages: Iterable[Any]
    ) -> Iterable[Any]:
        # Add write logic here
        for message in input_messages:
            # Process message
            yield message
    
    def check(self, logger: AirbyteLogger, config: Mapping[str, Any]) -> Any:
        try:
            # Add connection check logic here
            return {"status": "SUCCEEDED"}
        except Exception as e:
            return {"status": "FAILED", "message": str(e)}
EOF
        sed -i "s/CUSTOM_NAME/${name//[-_]/''}/g" "${connector_dir}/${type}_${name//-/_}/${type}.py"
    fi
    
    # Create Dockerfile
    cat > "${connector_dir}/Dockerfile" <<EOF
FROM airbyte/python-connector-base:1.1.0

WORKDIR /airbyte/integration_code
COPY . .

RUN pip install --upgrade pip && pip install .

ENV AIRBYTE_ENTRYPOINT "python /airbyte/integration_code/main.py"
ENTRYPOINT ["python", "/airbyte/integration_code/main.py"]

LABEL io.airbyte.version=0.1.0
LABEL io.airbyte.name=airbyte/${type}-${name}
EOF
    
    # Create main.py
    cat > "${connector_dir}/main.py" <<EOF
import sys
from ${type}_${name//-/_} import ${type^}${name//[-_]/''}

if __name__ == "__main__":
    ${type^}${name//[-_]/''}.run(sys.argv[1:])
EOF
    
    # Create requirements.txt
    cat > "${connector_dir}/requirements.txt" <<EOF
airbyte-cdk~=0.51.0
requests~=2.31.0
EOF
    
    # Create metadata.yaml
    cat > "${connector_dir}/metadata.yaml" <<EOF
data:
  connectorSubtype: api
  connectorType: ${type}
  definitionId: $(uuidgen || echo "00000000-0000-0000-0000-000000000000")
  dockerImageTag: 0.1.0
  dockerRepository: airbyte/${type}-${name}
  githubIssueLabel: ${type}-${name}
  icon: icon.svg
  license: MIT
  name: ${type^} ${name}
  registries:
    cloud:
      enabled: false
    oss:
      enabled: true
  releaseStage: alpha
  documentationUrl: https://docs.airbyte.com/integrations/${type}s/${name}
metadataSpecVersion: "1.0"
EOF
    
    log_info "CDK connector initialized at: $connector_dir"
    echo ""
    echo "Next steps:"
    echo "1. Edit ${connector_dir}/${type}_${name//-/_}/${type}.py to implement your connector logic"
    echo "2. Update ${connector_dir}/${type}_${name//-/_}/spec.yaml with connection parameters"
    echo "3. Build: vrooli resource airbyte cdk build ${type}-${name}"
    echo "4. Test: vrooli resource airbyte cdk test ${type}-${name}"
    echo "5. Deploy: vrooli resource airbyte cdk deploy ${type}-${name}"
    
    return 0
}

# Build connector Docker image
cdk_build() {
    local connector="${1:-}"
    
    if [[ -z "$connector" ]]; then
        log_error "Connector name required"
        return 1
    fi
    
    local connector_dir="${CDK_DIR}/${connector}"
    if [[ ! -d "$connector_dir" ]]; then
        log_error "Connector not found: $connector"
        return 1
    fi
    
    log_info "Building connector: ${connector}..."
    
    # Build Docker image
    cd "$connector_dir"
    docker build -t "airbyte/${connector}:dev" .
    
    log_info "Connector built successfully: airbyte/${connector}:dev"
    return 0
}

# Test connector
cdk_test() {
    local connector="${1:-}"
    
    if [[ -z "$connector" ]]; then
        log_error "Connector name required"
        return 1
    fi
    
    local connector_dir="${CDK_DIR}/${connector}"
    if [[ ! -d "$connector_dir" ]]; then
        log_error "Connector not found: $connector"
        return 1
    fi
    
    # Create test config
    local test_config="${connector_dir}/test_config.json"
    if [[ ! -f "$test_config" ]]; then
        cat > "$test_config" <<'EOF'
{
    "api_key": "test_key"
}
EOF
        log_info "Created test config at: $test_config"
        echo "Please update with actual test credentials"
        return 1
    fi
    
    log_info "Testing connector: ${connector}..."
    
    # Run connector tests
    docker run --rm \
        -v "${connector_dir}/test_config.json:/config.json" \
        "airbyte/${connector}:dev" \
        check --config /config.json
    
    local result=$?
    if [[ $result -eq 0 ]]; then
        log_info "✅ Connector test passed"
    else
        log_error "❌ Connector test failed"
    fi
    
    return $result
}

# Deploy connector to Airbyte
cdk_deploy() {
    local connector="${1:-}"
    
    if [[ -z "$connector" ]]; then
        log_error "Connector name required"
        return 1
    fi
    
    local connector_dir="${CDK_DIR}/${connector}"
    if [[ ! -d "$connector_dir" ]]; then
        log_error "Connector not found: $connector"
        return 1
    fi
    
    log_info "Deploying connector to Airbyte: ${connector}..."
    
    # Load the image into kind cluster for abctl deployment
    if docker ps | grep -q airbyte-abctl-control-plane; then
        log_info "Loading image into Kubernetes cluster..."
        kind load docker-image "airbyte/${connector}:dev" --name airbyte-abctl 2>/dev/null || {
            log_error "Failed to load image into cluster"
            return 1
        }
        
        log_info "✅ Connector deployed to cluster"
        echo ""
        echo "The custom connector is now available in your Airbyte instance."
        echo "Access the webapp at http://localhost:8002 to configure and use it."
    else
        log_error "Airbyte cluster not running. Start with: vrooli resource airbyte manage start"
        return 1
    fi
    
    return 0
}

# List CDK connectors
cdk_list() {
    log_info "Custom connectors in ${CDK_DIR}:"
    
    if [[ ! -d "$CDK_DIR" ]] || [[ -z "$(ls -A "$CDK_DIR" 2>/dev/null)" ]]; then
        echo "  (none)"
        return 0
    fi
    
    for dir in "$CDK_DIR"/*; do
        if [[ -d "$dir" ]]; then
            local name
            name=$(basename "$dir")
            local built="❌"
            if docker images | grep -q "airbyte/${name}"; then
                built="✅"
            fi
            echo "  - ${name} (Built: ${built})"
        fi
    done
    
    return 0
}

# Generate connector documentation
cdk_docs() {
    local connector="${1:-}"
    
    if [[ -z "$connector" ]]; then
        log_error "Connector name required"
        return 1
    fi
    
    local connector_dir="${CDK_DIR}/${connector}"
    if [[ ! -d "$connector_dir" ]]; then
        log_error "Connector not found: $connector"
        return 1
    fi
    
    log_info "Generating documentation for: ${connector}..."
    
    # Create docs directory
    mkdir -p "${connector_dir}/docs"
    
    # Generate README
    cat > "${connector_dir}/docs/README.md" <<EOF
# ${connector} Connector

## Overview
Custom Airbyte connector for ${connector}.

## Configuration
$(cat "${connector_dir}"/**/spec.yaml 2>/dev/null | grep -A20 "properties:" || echo "See spec.yaml for configuration details")

## Development
\`\`\`bash
# Build
vrooli resource airbyte cdk build ${connector}

# Test
vrooli resource airbyte cdk test ${connector}

# Deploy
vrooli resource airbyte cdk deploy ${connector}
\`\`\`

## API Reference
See implementation in ${connector_dir}
EOF
    
    log_info "Documentation generated at: ${connector_dir}/docs/README.md"
    return 0
}

# Main CDK command handler
cmd_cdk() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        init)
            cdk_init "$@"
            ;;
        build)
            cdk_build "$@"
            ;;
        test)
            cdk_test "$@"
            ;;
        deploy)
            cdk_deploy "$@"
            ;;
        list)
            cdk_list "$@"
            ;;
        docs)
            cdk_docs "$@"
            ;;
        *)
            echo "Usage: vrooli resource airbyte cdk <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  init <name> [source|destination]  Initialize new connector"
            echo "  build <connector>                  Build connector Docker image"
            echo "  test <connector>                   Test connector"
            echo "  deploy <connector>                 Deploy to Airbyte"
            echo "  list                              List custom connectors"
            echo "  docs <connector>                  Generate documentation"
            return 1
            ;;
    esac
}