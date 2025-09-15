#!/bin/bash

# Monitoring functions for Mail-in-a-Box resource

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MAILINABOX_MONITOR_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "$MAILINABOX_MONITOR_LIB_DIR/core.sh"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Monitor email queue
mailinabox_monitor_queue() {
    if ! mailinabox_is_running; then
        echo -e "${RED}✗${NC} Mail server is not running"
        return 1
    fi
    
    echo -e "${BOLD}${BLUE}📧 Email Queue Status${NC}"
    echo "------------------------"
    
    # Get queue information
    local queue_output=$(docker exec "$MAILINABOX_CONTAINER_NAME" postqueue -p 2>/dev/null)
    
    if [[ -z "$queue_output" ]] || echo "$queue_output" | grep -q "Mail queue is empty"; then
        echo -e "${GREEN}✓${NC} Queue is empty - all mail delivered"
    else
        local queue_count=$(echo "$queue_output" | grep -c "^[A-F0-9]")
        echo -e "${YELLOW}⚠${NC} $queue_count message(s) in queue"
        echo "$queue_output" | head -20
    fi
    
    echo ""
}

# Monitor email delivery statistics
mailinabox_monitor_stats() {
    if ! mailinabox_is_running; then
        echo -e "${RED}✗${NC} Mail server is not running"
        return 1
    fi
    
    echo -e "${BOLD}${BLUE}📊 Email Statistics (Last 24 hours)${NC}"
    echo "------------------------------------"
    
    # Get statistics from mail logs
    local sent=$(docker exec "$MAILINABOX_CONTAINER_NAME" sh -c 'grep "status=sent" /var/log/mail/mail.log 2>/dev/null | wc -l' || echo "0")
    local received=$(docker exec "$MAILINABOX_CONTAINER_NAME" sh -c 'grep "postfix/smtpd.*from=" /var/log/mail/mail.log 2>/dev/null | wc -l' || echo "0")
    local bounced=$(docker exec "$MAILINABOX_CONTAINER_NAME" sh -c 'grep "status=bounced" /var/log/mail/mail.log 2>/dev/null | wc -l' || echo "0")
    local deferred=$(docker exec "$MAILINABOX_CONTAINER_NAME" sh -c 'grep "status=deferred" /var/log/mail/mail.log 2>/dev/null | wc -l' || echo "0")
    
    echo -e "${CYAN}ℹ${NC} Sent: $sent messages"
    echo -e "${CYAN}ℹ${NC} Received: $received messages"
    [[ "$bounced" -gt 0 ]] && echo -e "${YELLOW}⚠${NC} Bounced: $bounced messages"
    [[ "$deferred" -gt 0 ]] && echo -e "${YELLOW}⚠${NC} Deferred: $deferred messages"
    
    echo ""
}

# Monitor service health
mailinabox_monitor_health() {
    echo -e "${BOLD}${BLUE}🏥 Service Health Check${NC}"
    echo "------------------------"
    
    # Check main container
    if mailinabox_is_running; then
        echo -e "${GREEN}✓${NC} Mail server container: Running"
        
        # Check SMTP
        if echo "QUIT" | timeout 5 nc "${MAILINABOX_BIND_ADDRESS}" "${MAILINABOX_PORT_SMTP}" 2>/dev/null | grep -q "220"; then
            echo -e "${GREEN}✓${NC} SMTP (port 25): Responding"
        else
            echo -e "${RED}✗${NC} SMTP (port 25): Not responding"
        fi
        
        # Check IMAP
        if timeout 5 openssl s_client -connect "${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_IMAPS}" -quiet 2>/dev/null | grep -q "OK"; then
            echo -e "${GREEN}✓${NC} IMAPS (port 993): Responding"
        else
            echo -e "${YELLOW}⚠${NC} IMAPS (port 993): May not be responding"
        fi
        
        # Check webmail if available
        if docker inspect mailinabox-webmail &>/dev/null && [[ "$(docker inspect -f '{{.State.Running}}' mailinabox-webmail 2>/dev/null)" == "true" ]]; then
            echo -e "${GREEN}✓${NC} Webmail container: Running"
            if curl -sf "http://${MAILINABOX_BIND_ADDRESS}:8080" >/dev/null; then
                echo -e "${GREEN}✓${NC} Webmail interface: Accessible"
            else
                echo -e "${YELLOW}⚠${NC} Webmail interface: Not accessible"
            fi
        else
            echo -e "${CYAN}ℹ${NC} Webmail: Not installed"
        fi
    else
        echo -e "${RED}✗${NC} Mail server container: Not running"
    fi
    
    echo ""
}

# Monitor disk usage
mailinabox_monitor_disk() {
    echo -e "${BOLD}${BLUE}💾 Disk Usage${NC}"
    echo "--------------"
    
    if [[ -d "$MAILINABOX_DATA_DIR" ]]; then
        local total_size=$(du -sh "$MAILINABOX_DATA_DIR" 2>/dev/null | cut -f1)
        echo -e "${CYAN}ℹ${NC} Total data directory size: ${total_size:-unknown}"
        
        # Break down by subdirectory
        if [[ -d "$MAILINABOX_MAIL_DIR" ]]; then
            local mail_size=$(du -sh "$MAILINABOX_MAIL_DIR" 2>/dev/null | cut -f1)
            echo -e "${CYAN}ℹ${NC}   Mail storage: ${mail_size:-0}"
        fi
        
        if [[ -d "$MAILINABOX_CONFIG_DIR" ]]; then
            local config_size=$(du -sh "$MAILINABOX_CONFIG_DIR" 2>/dev/null | cut -f1)
            echo -e "${CYAN}ℹ${NC}   Configuration: ${config_size:-0}"
        fi
    else
        echo -e "${YELLOW}⚠${NC} Data directory not found"
    fi
    
    echo ""
}

# Monitor recent errors
mailinabox_monitor_errors() {
    if ! mailinabox_is_running; then
        return 1
    fi
    
    echo -e "${BOLD}${BLUE}⚠️ Recent Errors (Last 10)${NC}"
    echo "---------------------------"
    
    local errors=$(docker exec "$MAILINABOX_CONTAINER_NAME" sh -c 'grep -E "(error|failed|fatal)" /var/log/mail/mail.log 2>/dev/null | tail -10')
    
    if [[ -n "$errors" ]]; then
        echo "$errors"
    else
        echo -e "${GREEN}✓${NC} No recent errors found"
    fi
    
    echo ""
}

# Main monitoring function
mailinabox_monitor() {
    local option="${1:-all}"
    
    case "$option" in
        queue)
            mailinabox_monitor_queue
            ;;
        stats)
            mailinabox_monitor_stats
            ;;
        health)
            mailinabox_monitor_health
            ;;
        disk)
            mailinabox_monitor_disk
            ;;
        errors)
            mailinabox_monitor_errors
            ;;
        all)
            mailinabox_monitor_health
            mailinabox_monitor_queue
            mailinabox_monitor_stats
            mailinabox_monitor_disk
            mailinabox_monitor_errors
            ;;
        *)
            echo -e "${RED}✗${NC} Unknown monitoring option: $option"
            echo "Available options: queue, stats, health, disk, errors, all"
            return 1
            ;;
    esac
}

# Continuous monitoring mode
mailinabox_monitor_watch() {
    local interval="${1:-30}"
    
    echo -e "${BOLD}${GREEN}Starting continuous monitoring (interval: ${interval}s)${NC}"
    echo -e "${CYAN}ℹ${NC} Press Ctrl+C to stop"
    echo ""
    
    while true; do
        clear
        echo -e "${BOLD}${CYAN}=== Mail-in-a-Box Monitor === $(date '+%Y-%m-%d %H:%M:%S')${NC}"
        echo ""
        mailinabox_monitor all
        sleep "$interval"
    done
}