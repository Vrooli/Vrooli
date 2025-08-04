#!/usr/bin/env bats
# Tests for Windmill install.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export WINDMILL_PORT="5681"
    export WINDMILL_CONTAINER_NAME="windmill-test"
    export WINDMILL_DB_CONTAINER_NAME="windmill-db-test"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_DB_PASSWORD="test-password"
    export WINDMILL_ADMIN_EMAIL="admin@test.com"
    export WINDMILL_ADMIN_PASSWORD="admin123"
    export WINDMILL_DATA_DIR="/tmp/windmill-test"
    export WINDMILL_IMAGE="windmillhq/windmill:latest"
    export YES="no"
    export FORCE="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    WINDMILL_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$WINDMILL_DATA_DIR"
    
    # Mock system functions
    
    system::check_port() {
        local port="$1"
        # Mock port as available unless specifically testing conflicts
        if [[ "$port" == "9999" ]]; then
            return 1  # Port busy for conflict tests
        fi
        return 0
    }
    
    system::get_available_port() {
        echo "5682"  # Mock alternative port
    }
    
    # Mock Docker functions
    
    # Mock curl
    
    # Mock openssl
    openssl() {
        case "$*" in
            *"rand"*)
                echo "randomly_generated_password_123"
                ;;
            *) echo "OPENSSL: $*" ;;
        esac
    }
    
    # Mock log functions
    
    # Mock Windmill utility functions
    windmill::container_exists() { return 1; }  # Default: no existing container
    windmill::is_running() { return 0; }
    windmill::is_healthy() { return 0; }
    windmill::create_network() { echo "Network created"; return 0; }
    windmill::create_volumes() { echo "Volumes created"; return 0; }
    windmill::start_database() { echo "Database started"; return 0; }
    windmill::start_container() { echo "Container started"; return 0; }
    
    # Mock resources functions
    resources::add_rollback_action() {
        echo "ROLLBACK_ACTION: $1 -> $2"
        return 0
    }
    
    # Load configuration and messages
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    windmill::export_config
    windmill::export_messages
    
    # Load the functions to test
    source "${WINDMILL_DIR}/lib/install.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$WINDMILL_DATA_DIR"
}

# Test installation prerequisites check
@test "windmill::check_prerequisites validates system requirements" {
    result=$(windmill::check_prerequisites)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Docker" ]]
}

# Test prerequisites failure
@test "windmill::check_prerequisites handles missing dependencies" {
    # Override to simulate missing Docker
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run windmill::check_prerequisites
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Docker" ]]
}

# Test directory creation
@test "windmill::create_directories creates required directories" {
    result=$(windmill::create_directories)
    
    [[ "$result" =~ "Creating" ]] || [[ "$result" =~ "directory" ]]
    [[ "$result" =~ "$WINDMILL_DATA_DIR" ]] || [[ "$result" =~ "data" ]]
}

# Test directory creation with existing directories
@test "windmill::create_directories handles existing directories" {
    # Create directories first
    mkdir -p "$WINDMILL_DATA_DIR"/{data,logs,backups}
    
    result=$(windmill::create_directories)
    
    [[ "$?" -eq 0 ]]
    [[ "$result" =~ "already exists" ]] || [[ "$result" =~ "directory" ]]
}

# Test port availability check
@test "windmill::check_port_availability validates port availability" {
    result=$(windmill::check_port_availability)
    
    [[ "$?" -eq 0 ]]
    [[ "$result" =~ "port" ]] || [[ "$result" =~ "available" ]]
}

# Test port conflict handling
@test "windmill::check_port_availability handles port conflicts" {
    export WINDMILL_CUSTOM_PORT="9999"  # Mock busy port
    
    run windmill::check_port_availability
    [ "$status" -eq 1 ]
    [[ "$output" =~ "port" ]]
    [[ "$output" =~ "in use" ]] || [[ "$output" =~ "busy" ]]
}

# Test Docker image pull
@test "windmill::pull_docker_images downloads container images" {
    result=$(windmill::pull_docker_images)
    
    [[ "$result" =~ "Pulling" ]] || [[ "$result" =~ "DOCKER_PULL" ]]
    [[ "$result" =~ "windmillhq/windmill" ]]
    [[ "$result" =~ "postgres" ]]
}

# Test Docker image pull failure
@test "windmill::pull_docker_images handles pull failure" {
    # Override docker to fail on pull
    docker() {
        case "$1" in
            "pull") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run windmill::pull_docker_images
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test environment variable generation
@test "windmill::generate_env_vars creates environment variables" {
    result=$(windmill::generate_env_vars)
    
    [[ "$result" =~ "environment" ]] || [[ "$result" =~ "variables" ]]
    [[ "$result" =~ "WINDMILL_DB_PASSWORD" ]] || [[ "$result" =~ "password" ]]
}

# Test configuration file creation
@test "windmill::create_config_files generates configuration files" {
    result=$(windmill::create_config_files)
    
    [[ "$result" =~ "config" ]] || [[ "$result" =~ "file" ]]
}

# Test database initialization
@test "windmill::initialize_database sets up database" {
    result=$(windmill::initialize_database)
    
    [[ "$result" =~ "database" ]]
    [[ "$result" =~ "initialized" ]] || [[ "$result" =~ "setup" ]]
}

# Test admin user creation
@test "windmill::create_admin_user creates initial admin user" {
    result=$(windmill::create_admin_user)
    
    [[ "$result" =~ "admin" ]] || [[ "$result" =~ "user" ]]
    [[ "$result" =~ "created" ]] || [[ "$result" =~ "admin@test.com" ]]
}

# Test workspace creation
@test "windmill::create_default_workspace creates initial workspace" {
    result=$(windmill::create_default_workspace)
    
    [[ "$result" =~ "workspace" ]]
    [[ "$result" =~ "created" ]] || [[ "$result" =~ "default" ]]
}

# Test full installation process
@test "windmill::install performs complete installation" {
    result=$(windmill::install)
    
    [[ "$result" =~ "Installing" ]]
    [[ "$result" =~ "Windmill" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "successful" ]]
}

# Test installation with existing containers
@test "windmill::install handles existing containers" {
    # Override to simulate existing container
    windmill::container_exists() { return 0; }
    
    result=$(windmill::install)
    
    [[ "$result" =~ "already" ]] || [[ "$result" =~ "exists" ]]
}

# Test force installation
@test "windmill::install performs force installation" {
    export FORCE="yes"
    windmill::container_exists() { return 0; }
    
    result=$(windmill::install)
    
    [[ "$result" =~ "force" ]] || [[ "$result" =~ "Installing" ]]
}

# Test Docker network setup
@test "windmill::setup_docker_network creates Docker network" {
    result=$(windmill::setup_docker_network)
    
    [[ "$result" =~ "network" ]]
    [[ "$result" =~ "DOCKER_NETWORK_CREATE:" ]] || [[ "$result" =~ "created" ]]
}

# Test Docker volume setup
@test "windmill::setup_docker_volumes creates Docker volumes" {
    result=$(windmill::setup_docker_volumes)
    
    [[ "$result" =~ "volume" ]]
    [[ "$result" =~ "DOCKER_VOLUME_CREATE:" ]] || [[ "$result" =~ "created" ]]
}

# Test health check after installation
@test "windmill::verify_installation checks installation health" {
    result=$(windmill::verify_installation)
    
    [[ "$result" =~ "verify" ]] || [[ "$result" =~ "health" ]]
    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "ready" ]]
}

# Test installation rollback
@test "windmill::rollback_installation cleans up failed installation" {
    result=$(windmill::rollback_installation)
    
    [[ "$result" =~ "rollback" ]] || [[ "$result" =~ "cleanup" ]]
}

# Test installation status reporting
@test "windmill::report_installation_status provides status information" {
    result=$(windmill::report_installation_status)
    
    [[ "$result" =~ "Windmill" ]]
    [[ "$result" =~ "http://localhost:5681" ]]
    [[ "$result" =~ "Web UI" ]] || [[ "$result" =~ "access" ]]
}

# Test SSL certificate setup
@test "windmill::setup_ssl_certificates configures SSL certificates" {
    result=$(windmill::setup_ssl_certificates)
    
    [[ "$result" =~ "SSL" ]] || [[ "$result" =~ "certificate" ]]
}

# Test backup creation during installation
@test "windmill::create_installation_backup creates backup before install" {
    result=$(windmill::create_installation_backup)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "created" ]]
}

# Test disk space check
@test "windmill::check_disk_space validates available disk space" {
    result=$(windmill::check_disk_space)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "space" ]]
}

# Test memory requirements check
@test "windmill::check_memory_requirements validates system memory" {
    result=$(windmill::check_memory_requirements)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "RAM" ]]
}

# Test network connectivity check
@test "windmill::check_network_connectivity validates internet access" {
    result=$(windmill::check_network_connectivity)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "connectivity" ]]
}

# Test Docker daemon check
@test "windmill::check_docker_daemon validates Docker service" {
    result=$(windmill::check_docker_daemon)
    
    [[ "$result" =~ "Docker" ]] || [[ "$result" =~ "daemon" ]]
}

# Test installation preparation
@test "windmill::prepare_installation sets up installation environment" {
    result=$(windmill::prepare_installation)
    
    [[ "$result" =~ "prepare" ]] || [[ "$result" =~ "setup" ]]
}

# Test installation cleanup
@test "windmill::cleanup_installation removes temporary files" {
    result=$(windmill::cleanup_installation)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "clean" ]]
}

# Test installation progress tracking
@test "windmill::track_installation_progress monitors installation steps" {
    result=$(windmill::track_installation_progress "Setting up database")
    
    [[ "$result" =~ "progress" ]] || [[ "$result" =~ "Setting up database" ]]
}

# Test installation error recovery
@test "windmill::recover_from_installation_error handles installation failures" {
    result=$(windmill::recover_from_installation_error "Docker pull failed")
    
    [[ "$result" =~ "recover" ]] || [[ "$result" =~ "error" ]]
}

# Test post-installation configuration
@test "windmill::post_install_configuration configures installed instance" {
    result=$(windmill::post_install_configuration)
    
    [[ "$result" =~ "post" ]] || [[ "$result" =~ "configuration" ]]
}

# Test installation summary
@test "windmill::generate_installation_summary creates installation report" {
    result=$(windmill::generate_installation_summary)
    
    [[ "$result" =~ "summary" ]] || [[ "$result" =~ "report" ]]
    [[ "$result" =~ "Windmill" ]]
}

# Test Docker Compose installation
@test "windmill::install_with_compose uses Docker Compose for installation" {
    result=$(windmill::install_with_compose)
    
    [[ "$result" =~ "compose" ]] || [[ "$result" =~ "docker-compose" ]]
}

# Test environment file creation
@test "windmill::create_env_file generates environment file" {
    result=$(windmill::create_env_file)
    
    [[ "$result" =~ "environment" ]] || [[ "$result" =~ ".env" ]]
}

# Test configuration validation
@test "windmill::validate_installation_config validates installation configuration" {
    result=$(windmill::validate_installation_config)
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "configuration" ]]
}

# Test dependency installation
@test "windmill::install_dependencies installs required dependencies" {
    result=$(windmill::install_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "install" ]]
}

# Test service registration
@test "windmill::register_service registers Windmill as system service" {
    result=$(windmill::register_service)
    
    [[ "$result" =~ "service" ]] || [[ "$result" =~ "register" ]]
}

# Test security configuration
@test "windmill::configure_security sets up security configuration" {
    result=$(windmill::configure_security)
    
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "secure" ]]
}

# Test firewall configuration
@test "windmill::configure_firewall configures firewall rules" {
    result=$(windmill::configure_firewall)
    
    [[ "$result" =~ "firewall" ]] || [[ "$result" =~ "port" ]]
}

# Test log rotation setup
@test "windmill::setup_log_rotation configures log rotation" {
    result=$(windmill::setup_log_rotation)
    
    [[ "$result" =~ "log" ]] || [[ "$result" =~ "rotation" ]]
}

# Test monitoring setup
@test "windmill::setup_monitoring configures monitoring and metrics" {
    result=$(windmill::setup_monitoring)
    
    [[ "$result" =~ "monitoring" ]] || [[ "$result" =~ "metrics" ]]
}

# Test update mechanism setup
@test "windmill::setup_update_mechanism configures automatic updates" {
    result=$(windmill::setup_update_mechanism)
    
    [[ "$result" =~ "update" ]] || [[ "$result" =~ "automatic" ]]
}

# Test installation verification
@test "windmill::verify_complete_installation performs comprehensive verification" {
    result=$(windmill::verify_complete_installation)
    
    [[ "$result" =~ "verify" ]] || [[ "$result" =~ "complete" ]]
    [[ "$result" =~ "installation" ]]
}