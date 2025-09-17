#!/usr/bin/env bash
################################################################################
# Zigbee2MQTT CLI Interface
# 
# Provides command-line interface for managing Zigbee2MQTT bridge
################################################################################

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    ZIGBEE2MQTT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${ZIGBEE2MQTT_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="zigbee2mqtt"

# Source common libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/exit_codes.sh"

# Load resource libraries
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

################################################################################
# Command Definitions
################################################################################

# Help command
show_help() {
    cat << EOF
Zigbee2MQTT Resource CLI - Zigbee to MQTT Bridge

USAGE:
    resource-zigbee2mqtt <command> [options]

COMMANDS:
    help                    Show this help message
    info                    Display resource information and configuration
    status                  Show current Zigbee2MQTT status
    logs                    View Zigbee2MQTT logs
    
    manage <action>         Lifecycle management:
        install            Install Zigbee2MQTT and dependencies
        start             Start Zigbee2MQTT service
        stop              Stop Zigbee2MQTT service
        restart           Restart Zigbee2MQTT service
        uninstall         Remove Zigbee2MQTT and cleanup
    
    test <suite>           Run tests:
        smoke             Quick health check (<30s)
        integration       Full integration tests (<120s)
        unit              Unit tests (<60s)
        all               Run all test suites
    
    content <action>       Manage device configurations:
        list              List paired devices
        add               Pair new device
        get <device>      Get device configuration
        remove <device>   Remove device from network
        execute <cmd>     Execute Zigbee2MQTT command
    
    device <action>        Device management:
        pair              Enable pairing mode
        unpair <device>   Remove device
        rename <old> <new> Rename device
        control <dev> <state> Set device on/off/toggle
        brightness <dev> <val> Set brightness (0-255)
        color <dev> <color>   Set color (hex or RGB)
        temperature <dev> <K> Set color temperature
        configure <device> Update device settings
    
    network <action>       Network management:
        map               Show network topology
        backup            Backup coordinator
        restore <file>    Restore coordinator backup
        channel <num>     Change Zigbee channel
    
    homeassistant <action> Home Assistant integration:
        discovery <cmd>   Enable/disable/status MQTT discovery
    
    ota <action>           Firmware updates:
        check [device]    Check for available updates
        update <device>   Update device firmware
    
    group <action>         Group management:
        create <name> <devices...> Create device group
        list              List all groups
        control <name> <state> Control group state
        remove <name>     Remove group
    
    scene <action>         Scene management:
        create <name> <group> Create scene from current state
        recall <name>     Activate scene
    
    touchlink <action>     Touchlink commissioning:
        scan              Scan for nearby Touchlink devices
        identify <ieee> [secs] Make device blink/flash
        reset <ieee>      Factory reset device via Touchlink
    
    converter <action>     External device converter management:
        add <file>        Add external converter file
        list              List installed converters
        remove <file>     Remove external converter
        generate <model> <vendor> Generate converter template

EXAMPLES:
    resource-zigbee2mqtt manage start       # Start Zigbee2MQTT
    resource-zigbee2mqtt device pair        # Enable pairing mode
    resource-zigbee2mqtt content list       # List all devices
    resource-zigbee2mqtt test smoke         # Run quick health check

ENVIRONMENT:
    ZIGBEE2MQTT_PORT         API port (default: 8090)
    MQTT_HOST                MQTT broker host (default: localhost)
    MQTT_PORT                MQTT broker port (default: 1883)
    ZIGBEE_ADAPTER           USB adapter path (default: auto-detect)

For more information, see: resources/zigbee2mqtt/README.md
EOF
}

# Manage commands
handle_manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            zigbee2mqtt::install "$@"
            ;;
        start)
            zigbee2mqtt::start "$@"
            ;;
        stop)
            zigbee2mqtt::stop "$@"
            ;;
        restart)
            zigbee2mqtt::restart "$@"
            ;;
        uninstall)
            zigbee2mqtt::uninstall "$@"
            ;;
        *)
            log::error "Unknown manage action: $action"
            echo "Valid actions: install, start, stop, restart, uninstall"
            return 1
            ;;
    esac
}

# Test commands
handle_test() {
    local suite="${1:-all}"
    shift || true
    
    case "$suite" in
        smoke)
            zigbee2mqtt::test::smoke "$@"
            ;;
        integration)
            zigbee2mqtt::test::integration "$@"
            ;;
        unit)
            zigbee2mqtt::test::unit "$@"
            ;;
        all)
            zigbee2mqtt::test::all "$@"
            ;;
        *)
            log::error "Unknown test suite: $suite"
            echo "Valid suites: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

# Content commands
handle_content() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        list)
            zigbee2mqtt::content::list "$@"
            ;;
        add)
            zigbee2mqtt::content::add "$@"
            ;;
        get)
            zigbee2mqtt::content::get "$@"
            ;;
        remove)
            zigbee2mqtt::content::remove "$@"
            ;;
        execute)
            zigbee2mqtt::content::execute "$@"
            ;;
        *)
            log::error "Unknown content action: $action"
            echo "Valid actions: list, add, get, remove, execute"
            return 1
            ;;
    esac
}

# Device commands
handle_device() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        pair)
            zigbee2mqtt::device::pair "$@"
            ;;
        unpair)
            zigbee2mqtt::device::unpair "$@"
            ;;
        rename)
            zigbee2mqtt::device::rename "$@"
            ;;
        control)
            zigbee2mqtt::device::control "$@"
            ;;
        brightness)
            zigbee2mqtt::device::brightness "$@"
            ;;
        color)
            zigbee2mqtt::device::color "$@"
            ;;
        temperature)
            zigbee2mqtt::device::temperature "$@"
            ;;
        configure)
            zigbee2mqtt::device::configure "$@"
            ;;
        *)
            log::error "Unknown device action: $action"
            echo "Valid actions: pair, unpair, rename, control, brightness, color, temperature, configure"
            return 1
            ;;
    esac
}

# HomeAssistant commands
handle_homeassistant() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        discovery)
            zigbee2mqtt::homeassistant::discovery "$@"
            ;;
        *)
            log::error "Unknown HomeAssistant action: $action"
            echo "Valid actions: discovery"
            return 1
            ;;
    esac
}

# OTA commands
handle_ota() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        check)
            zigbee2mqtt::ota::check "$@"
            ;;
        update)
            zigbee2mqtt::ota::update "$@"
            ;;
        *)
            log::error "Unknown OTA action: $action"
            echo "Valid actions: check, update"
            return 1
            ;;
    esac
}

# Group commands
handle_group() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        create)
            zigbee2mqtt::group::create "$@"
            ;;
        list)
            zigbee2mqtt::group::list "$@"
            ;;
        control)
            zigbee2mqtt::group::control "$@"
            ;;
        remove)
            zigbee2mqtt::group::remove "$@"
            ;;
        *)
            log::error "Unknown group action: $action"
            echo "Valid actions: create, list, control, remove"
            return 1
            ;;
    esac
}

# Scene commands
handle_scene() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        create)
            zigbee2mqtt::scene::create "$@"
            ;;
        recall)
            zigbee2mqtt::scene::recall "$@"
            ;;
        *)
            log::error "Unknown scene action: $action"
            echo "Valid actions: create, recall"
            return 1
            ;;
    esac
}

# Network commands
handle_network() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        map)
            zigbee2mqtt::network::map "$@"
            ;;
        backup)
            zigbee2mqtt::network::backup "$@"
            ;;
        restore)
            zigbee2mqtt::network::restore "$@"
            ;;
        channel)
            zigbee2mqtt::network::channel "$@"
            ;;
        *)
            log::error "Unknown network action: $action"
            echo "Valid actions: map, backup, restore, channel"
            return 1
            ;;
    esac
}

# Touchlink commands
handle_touchlink() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        scan)
            zigbee2mqtt::touchlink::scan "$@"
            ;;
        identify)
            zigbee2mqtt::touchlink::identify "$@"
            ;;
        reset)
            zigbee2mqtt::touchlink::reset "$@"
            ;;
        *)
            log::error "Unknown touchlink action: $action"
            echo "Valid actions: scan, identify, reset"
            return 1
            ;;
    esac
}

# Converter commands
handle_converter() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        add)
            zigbee2mqtt::converter::add "$@"
            ;;
        list)
            zigbee2mqtt::converter::list "$@"
            ;;
        remove)
            zigbee2mqtt::converter::remove "$@"
            ;;
        generate)
            zigbee2mqtt::converter::generate "$@"
            ;;
        *)
            log::error "Unknown converter action: $action"
            echo "Valid actions: add, list, remove, generate"
            return 1
            ;;
    esac
}

################################################################################
# Main Execution
################################################################################

main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help|--help|-h)
            show_help
            ;;
        info)
            zigbee2mqtt::info
            ;;
        status)
            zigbee2mqtt::status "$@"
            ;;
        logs)
            zigbee2mqtt::logs "$@"
            ;;
        manage)
            handle_manage "$@"
            ;;
        test)
            handle_test "$@"
            ;;
        content)
            handle_content "$@"
            ;;
        device)
            handle_device "$@"
            ;;
        network)
            handle_network "$@"
            ;;
        homeassistant|ha)
            handle_homeassistant "$@"
            ;;
        ota)
            handle_ota "$@"
            ;;
        group)
            handle_group "$@"
            ;;
        scene)
            handle_scene "$@"
            ;;
        touchlink)
            handle_touchlink "$@"
            ;;
        converter)
            handle_converter "$@"
            ;;
        *)
            log::error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"