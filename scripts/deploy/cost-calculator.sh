#!/usr/bin/env bash
# DigitalOcean Kubernetes Cost Calculator for Vrooli
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

cost::header() {
    echo -e "${BLUE}$1${NC}"
    echo "=============================================="
}

cost::info() {
    echo -e "${GREEN}$1${NC}"
}

cost::warning() {
    echo -e "${YELLOW}$1${NC}"
}

cost::calculate() {
    local node_size=$1
    local node_count=$2
    local include_databases=$3
    local include_load_balancer=$4
    
    # Node pricing per month
    local node_cost_per_month=0
    case $node_size in
        "s-1vcpu-2gb")
            node_cost_per_month=12
            ;;
        "s-2vcpu-4gb")
            node_cost_per_month=24
            ;;
        "s-4vcpu-8gb")
            node_cost_per_month=48
            ;;
        "c-2")
            node_cost_per_month=48
            ;;
        "c-4")
            node_cost_per_month=96
            ;;
        *)
            echo "Unknown node size: $node_size"
            return 1
            ;;
    esac
    
    local total_node_cost=$((node_cost_per_month * node_count))
    local database_cost=0
    local load_balancer_cost=0
    
    if [[ "$include_databases" == "true" ]]; then
        database_cost=30  # $15 PostgreSQL + $15 Redis
    fi
    
    if [[ "$include_load_balancer" == "true" ]]; then
        load_balancer_cost=12
    fi
    
    local total_cost=$((total_node_cost + database_cost + load_balancer_cost))
    
    echo "Node Cost: \$${total_node_cost}"
    echo "Database Cost: \$${database_cost}"
    echo "Load Balancer Cost: \$${load_balancer_cost}"
    echo "TOTAL: \$${total_cost}/month"
    
    return $total_cost
}

cost::show_configurations() {
    cost::header "DigitalOcean Kubernetes Cost Calculator"
    
    echo "Node Size Options:"
    echo "=================="
    echo "s-1vcpu-2gb:  1 vCPU,  2GB RAM,  50GB SSD  (\$12/month each)"
    echo "s-2vcpu-4gb:  2 vCPU,  4GB RAM,  80GB SSD  (\$24/month each)"
    echo "s-4vcpu-8gb:  4 vCPU,  8GB RAM, 160GB SSD  (\$48/month each)"
    echo "c-2:          2 vCPU,  4GB RAM,  50GB SSD  (\$48/month each, CPU-optimized)"
    echo "c-4:          4 vCPU,  8GB RAM, 100GB SSD  (\$96/month each, CPU-optimized)"
    echo ""
    
    echo "Additional Services:"
    echo "==================="
    echo "Load Balancer:     \$12/month"
    echo "PostgreSQL (1GB):  \$15/month"
    echo "Redis (1GB):       \$15/month"
    echo "Block Storage:     \$0.10/GB/month"
    echo ""
    
    cost::header "Configuration Examples"
    
    echo ""
    cost::info "💡 Minimal Setup (Development/Testing)"
    echo "3 × s-1vcpu-2gb nodes + Load Balancer + Databases"
    cost::calculate "s-1vcpu-2gb" 3 true true
    total_minimal=$?
    
    echo ""
    cost::info "⭐ Recommended Setup (Production)"
    echo "3 × s-2vcpu-4gb nodes + Load Balancer + Databases"
    cost::calculate "s-2vcpu-4gb" 3 true true
    total_recommended=$?
    
    echo ""
    cost::info "🚀 High Performance Setup"
    echo "3 × s-4vcpu-8gb nodes + Load Balancer + Databases"
    cost::calculate "s-4vcpu-8gb" 3 true true
    total_high_perf=$?
    
    echo ""
    cost::info "💰 Budget Option (Self-managed DB)"
    echo "3 × s-1vcpu-2gb nodes + Load Balancer (no managed databases)"
    cost::calculate "s-1vcpu-2gb" 3 false true
    total_budget=$?
    
    echo ""
    cost::header "Annual Cost Comparison"
    printf "%-25s %10s %12s\n" "Configuration" "Monthly" "Annual"
    printf "%-25s %10s %12s\n" "=============" "=======" "======"
    printf "%-25s %9s%s %11s%s\n" "Minimal Setup" "\$" "${total_minimal}" "\$" "$((total_minimal * 12))"
    printf "%-25s %9s%s %11s%s\n" "Recommended" "\$" "${total_recommended}" "\$" "$((total_recommended * 12))"
    printf "%-25s %9s%s %11s%s\n" "High Performance" "\$" "${total_high_perf}" "\$" "$((total_high_perf * 12))"
    printf "%-25s %9s%s %11s%s\n" "Budget Option" "\$" "${total_budget}" "\$" "$((total_budget * 12))"
    
    echo ""
    cost::header "Cost Optimization Tips"
    echo "✅ Start with the Recommended setup - you can always scale up"
    echo "✅ Use horizontal pod autoscaling to handle traffic spikes efficiently"
    echo "✅ Consider DigitalOcean Reserved Instances (20% discount for 1-year commitment)"
    echo "✅ Monitor resource usage and right-size your nodes based on actual usage"
    echo "✅ Use managed databases for production (worth the cost for reliability)"
    echo "✅ Enable automatic node upgrades to get latest features and security patches"
    
    echo ""
    cost::header "Traffic Capacity Estimates"
    echo "Minimal Setup:     ~1,000-5,000 daily active users"
    echo "Recommended:       ~10,000-50,000 daily active users"
    echo "High Performance:  ~100,000+ daily active users"
    echo ""
    echo "Note: Actual capacity depends on your application's resource usage patterns"
}

cost::interactive_calculator() {
    cost::header "Interactive Cost Calculator"
    
    echo "Configure your setup:"
    echo ""
    
    # Get node size
    echo "Select node size:"
    echo "1) s-1vcpu-2gb  (\$12/month each)"
    echo "2) s-2vcpu-4gb  (\$24/month each) [Recommended]"
    echo "3) s-4vcpu-8gb  (\$48/month each)"
    echo "4) c-2          (\$48/month each, CPU-optimized)"
    echo "5) c-4          (\$96/month each, CPU-optimized)"
    read -p "Choice (1-5): " -r size_choice
    
    case $size_choice in
        1) node_size="s-1vcpu-2gb" ;;
        2) node_size="s-2vcpu-4gb" ;;
        3) node_size="s-4vcpu-8gb" ;;
        4) node_size="c-2" ;;
        5) node_size="c-4" ;;
        *) echo "Invalid choice, using recommended s-2vcpu-4gb"; node_size="s-2vcpu-4gb" ;;
    esac
    
    # Get node count
    read -p "Number of nodes (3-10, recommended: 3): " -r node_count
    node_count=${node_count:-3}
    
    # Validate node count
    if [[ $node_count -lt 3 ]]; then
        echo "Warning: Less than 3 nodes reduces high availability"
        node_count=3
    elif [[ $node_count -gt 10 ]]; then
        echo "Warning: More than 10 nodes may be overkill for most applications"
        node_count=10
    fi
    
    # Get database preference
    read -p "Include managed databases? (Y/n): " -r db_choice
    include_databases="true"
    if [[ "$db_choice" =~ ^[Nn]$ ]]; then
        include_databases="false"
    fi
    
    # Get load balancer preference
    read -p "Include load balancer? (Y/n): " -r lb_choice
    include_load_balancer="true"
    if [[ "$lb_choice" =~ ^[Nn]$ ]]; then
        include_load_balancer="false"
    fi
    
    echo ""
    cost::header "Your Custom Configuration"
    echo "Node Size: $node_size"
    echo "Node Count: $node_count"
    echo "Managed Databases: $include_databases"
    echo "Load Balancer: $include_load_balancer"
    echo ""
    
    cost::calculate "$node_size" "$node_count" "$include_databases" "$include_load_balancer"
    local total_cost=$?
    
    echo ""
    echo "Annual cost: \$$(($total_cost * 12))"
    
    if [[ $total_cost -lt 60 ]]; then
        cost::warning "💡 This is a budget configuration. Consider upgrading for production workloads."
    elif [[ $total_cost -lt 150 ]]; then
        cost::info "✅ This is a good production configuration."
    else
        cost::warning "💰 This is a high-performance configuration. Make sure you need this capacity."
    fi
}

cost::main() {
    case "${1:-show}" in
        "show")
            cost::show_configurations
            ;;
        "interactive")
            cost::interactive_calculator
            ;;
        "help")
            echo "Usage: $0 [show|interactive|help]"
            echo "  show        - Show predefined configurations (default)"
            echo "  interactive - Interactive cost calculator"
            echo "  help        - Show this help"
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run the calculator
cost::main "$@"