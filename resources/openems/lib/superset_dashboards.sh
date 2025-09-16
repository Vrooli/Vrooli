#!/usr/bin/env bash

# Apache Superset Dashboard Integration for OpenEMS
# Creates energy analytics dashboards and visualizations

set -euo pipefail

# Source required libraries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh" 2>/dev/null || true
source "${SCRIPT_DIR}/../config/defaults.sh" 2>/dev/null || true

# Define logging functions if not available
if ! command -v log_info &> /dev/null; then
    log_info() { echo "[INFO] $*"; }
    log_success() { echo "[SUCCESS] ✅ $*"; }
    log_warn() { echo "[WARN] ⚠️  $*"; }
    log_error() { echo "[ERROR] ❌ $*"; }
fi

# Superset configuration
SUPERSET_PORT="${SUPERSET_PORT:-8088}"
SUPERSET_URL="http://localhost:${SUPERSET_PORT}"

# QuestDB configuration for data source
QUESTDB_PORT="${QUESTDB_PORT:-9010}"
QUESTDB_URL="http://localhost:${QUESTDB_PORT}"

# -------------------------------
# Dashboard Templates
# -------------------------------

create_energy_overview_dashboard() {
    log_info "Creating Energy Overview Dashboard template"
    
    cat > "/tmp/energy-overview-dashboard.json" <<'EOF'
{
    "dashboard_title": "OpenEMS Energy Overview",
    "description": "Real-time energy monitoring and analytics",
    "css": "",
    "slug": "openems-energy-overview",
    "published": true,
    "position_json": {
        "CHART-solar-production": {
            "type": "CHART",
            "id": "solar-production",
            "children": [],
            "meta": {
                "width": 6,
                "height": 50,
                "chartId": 1,
                "sliceName": "Solar Production"
            }
        },
        "CHART-battery-soc": {
            "type": "CHART",
            "id": "battery-soc",
            "children": [],
            "meta": {
                "width": 6,
                "height": 50,
                "chartId": 2,
                "sliceName": "Battery State of Charge"
            }
        },
        "CHART-grid-power": {
            "type": "CHART",
            "id": "grid-power",
            "children": [],
            "meta": {
                "width": 12,
                "height": 50,
                "chartId": 3,
                "sliceName": "Grid Power Flow"
            }
        },
        "CHART-consumption": {
            "type": "CHART",
            "id": "consumption",
            "children": [],
            "meta": {
                "width": 6,
                "height": 50,
                "chartId": 4,
                "sliceName": "Energy Consumption"
            }
        },
        "CHART-efficiency": {
            "type": "CHART",
            "id": "efficiency",
            "children": [],
            "meta": {
                "width": 6,
                "height": 50,
                "chartId": 5,
                "sliceName": "System Efficiency"
            }
        }
    }
}
EOF
    
    log_info "Energy overview dashboard template created at /tmp/energy-overview-dashboard.json"
    return 0
}

create_solar_analytics_dashboard() {
    log_info "Creating Solar Analytics Dashboard template"
    
    cat > "/tmp/solar-analytics-dashboard.json" <<'EOF'
{
    "dashboard_title": "Solar Analytics",
    "description": "Detailed solar generation analytics and forecasting",
    "slug": "openems-solar-analytics",
    "published": true,
    "charts": [
        {
            "slice_name": "Daily Solar Generation",
            "viz_type": "line",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["solar_power"],
                "groupby": ["hour"],
                "time_range": "Last 24 hours",
                "line_interpolation": "linear",
                "show_legend": true,
                "y_axis_label": "Power (kW)",
                "x_axis_label": "Time"
            }
        },
        {
            "slice_name": "Solar vs Consumption",
            "viz_type": "area",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["solar_power", "consumption_power"],
                "groupby": ["timestamp"],
                "time_range": "Last 7 days",
                "show_legend": true,
                "stack": false,
                "y_axis_label": "Power (kW)"
            }
        },
        {
            "slice_name": "Solar Efficiency Heatmap",
            "viz_type": "heatmap",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["solar_efficiency"],
                "all_columns_x": ["hour"],
                "all_columns_y": ["day_of_week"],
                "linear_color_scheme": "green_yellow_red"
            }
        },
        {
            "slice_name": "Monthly Solar Yield",
            "viz_type": "bar",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["total_solar_energy"],
                "groupby": ["month"],
                "time_range": "Last 12 months",
                "y_axis_label": "Energy (kWh)"
            }
        }
    ]
}
EOF
    
    log_info "Solar analytics dashboard template created at /tmp/solar-analytics-dashboard.json"
    return 0
}

create_battery_management_dashboard() {
    log_info "Creating Battery Management Dashboard template"
    
    cat > "/tmp/battery-management-dashboard.json" <<'EOF'
{
    "dashboard_title": "Battery Management",
    "description": "Battery storage monitoring and optimization",
    "slug": "openems-battery-management",
    "published": true,
    "charts": [
        {
            "slice_name": "Battery SOC Gauge",
            "viz_type": "gauge_chart",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metric": "battery_soc",
                "max_value": 100,
                "intervals": [0, 20, 80, 100],
                "interval_colors": ["danger", "warning", "success"],
                "value_format": ".1f"
            }
        },
        {
            "slice_name": "Charge/Discharge Power",
            "viz_type": "time_series_bar",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["battery_power"],
                "groupby": ["timestamp"],
                "time_range": "Last 24 hours",
                "y_axis_label": "Power (kW)",
                "color_scheme": "diverging_red_green"
            }
        },
        {
            "slice_name": "Battery Cycles",
            "viz_type": "big_number_total",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metric": "battery_cycles",
                "subheader": "Total charge/discharge cycles"
            }
        },
        {
            "slice_name": "Battery Health",
            "viz_type": "line",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["battery_health", "battery_temperature"],
                "groupby": ["date"],
                "time_range": "Last 30 days",
                "y_axis_label": "Health (%) / Temperature (°C)",
                "show_legend": true
            }
        }
    ]
}
EOF
    
    log_info "Battery management dashboard template created at /tmp/battery-management-dashboard.json"
    return 0
}

create_grid_interaction_dashboard() {
    log_info "Creating Grid Interaction Dashboard template"
    
    cat > "/tmp/grid-interaction-dashboard.json" <<'EOF'
{
    "dashboard_title": "Grid Interaction",
    "description": "Grid power flow and peak demand management",
    "slug": "openems-grid-interaction",
    "published": true,
    "charts": [
        {
            "slice_name": "Grid Power Flow",
            "viz_type": "time_series_line",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["grid_import", "grid_export"],
                "groupby": ["timestamp"],
                "time_range": "Last 48 hours",
                "y_axis_label": "Power (kW)",
                "show_legend": true,
                "line_interpolation": "step-after"
            }
        },
        {
            "slice_name": "Peak Demand",
            "viz_type": "big_number",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metric": "peak_demand_today",
                "compare_lag": 1,
                "compare_suffix": "d",
                "subheader": "Today's peak demand"
            }
        },
        {
            "slice_name": "Energy Costs",
            "viz_type": "mixed_timeseries",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["energy_cost", "electricity_tariff"],
                "groupby": ["hour"],
                "time_range": "Last 7 days",
                "y_axis_label": "Cost ($/kWh)",
                "y_axis_2_label": "Tariff ($/kWh)"
            }
        },
        {
            "slice_name": "Self-Sufficiency Rate",
            "viz_type": "pie",
            "datasource_name": "QuestDB OpenEMS",
            "params": {
                "metrics": ["self_consumed", "grid_imported"],
                "groupby": ["energy_source"],
                "time_range": "Last 30 days",
                "donut": true,
                "show_legend": true
            }
        }
    ]
}
EOF
    
    log_info "Grid interaction dashboard template created at /tmp/grid-interaction-dashboard.json"
    return 0
}

# -------------------------------
# SQL Queries for QuestDB
# -------------------------------

create_questdb_views() {
    log_info "Creating QuestDB views for Superset integration"
    
    cat > "/tmp/questdb-openems-views.sql" <<'EOF'
-- Create aggregated views for Superset dashboards

-- Hourly energy aggregates
CREATE TABLE IF NOT EXISTS energy_hourly AS (
    SELECT 
        timestamp_floor('h', timestamp) as hour,
        avg(solar_power) as avg_solar,
        avg(battery_power) as avg_battery,
        avg(grid_power) as avg_grid,
        avg(consumption_power) as avg_consumption,
        sum(solar_energy) as total_solar,
        sum(consumption_energy) as total_consumption
    FROM openems_telemetry
    SAMPLE BY 1h
);

-- Daily energy summary
CREATE TABLE IF NOT EXISTS energy_daily AS (
    SELECT
        timestamp_floor('d', timestamp) as day,
        max(solar_power) as peak_solar,
        min(battery_soc) as min_battery_soc,
        max(battery_soc) as max_battery_soc,
        max(grid_power) as peak_demand,
        sum(solar_energy) as daily_solar_yield,
        sum(consumption_energy) as daily_consumption,
        sum(CASE WHEN grid_power > 0 THEN grid_energy ELSE 0 END) as daily_import,
        sum(CASE WHEN grid_power < 0 THEN abs(grid_energy) ELSE 0 END) as daily_export
    FROM openems_telemetry
    SAMPLE BY 1d
);

-- Real-time KPIs
CREATE TABLE IF NOT EXISTS energy_kpis AS (
    SELECT
        timestamp,
        solar_power / NULLIF(consumption_power, 0) as self_sufficiency_ratio,
        battery_soc,
        CASE 
            WHEN grid_power > 15 THEN 'peak'
            WHEN grid_power > 10 THEN 'high'
            WHEN grid_power > 5 THEN 'medium'
            ELSE 'low'
        END as demand_level,
        solar_power * 0.95 as solar_efficiency,
        battery_power / NULLIF(battery_capacity, 0) as c_rate
    FROM openems_telemetry
    WHERE timestamp > dateadd('h', -24, now())
);

-- Battery cycle counting
CREATE TABLE IF NOT EXISTS battery_cycles AS (
    SELECT
        timestamp_floor('d', timestamp) as day,
        sum(abs(battery_power) * 0.25 / battery_capacity) as daily_cycles,
        avg(battery_temperature) as avg_temperature,
        avg(battery_health) as avg_health
    FROM openems_telemetry
    WHERE battery_power != 0
    SAMPLE BY 1d
);
EOF
    
    log_info "QuestDB views SQL created at /tmp/questdb-openems-views.sql"
    return 0
}

# -------------------------------
# Superset Configuration
# -------------------------------

create_superset_dataset_config() {
    log_info "Creating Superset dataset configuration"
    
    cat > "/tmp/superset-openems-dataset.yaml" <<'EOF'
database_name: QuestDB OpenEMS
sqlalchemy_uri: postgresql://admin:quest@localhost:8812/qdb
tables:
  - table_name: openems_telemetry
    schema: public
    columns:
      - column_name: timestamp
        type: TIMESTAMP
        is_dttm: true
      - column_name: solar_power
        type: FLOAT
        description: Solar generation power in kW
      - column_name: battery_power
        type: FLOAT
        description: Battery charge/discharge power in kW
      - column_name: battery_soc
        type: FLOAT
        description: Battery state of charge in %
      - column_name: grid_power
        type: FLOAT
        description: Grid import/export power in kW
      - column_name: consumption_power
        type: FLOAT
        description: Total consumption power in kW
    metrics:
      - metric_name: avg_solar_power
        expression: AVG(solar_power)
        description: Average solar power
      - metric_name: max_grid_power
        expression: MAX(grid_power)
        description: Peak grid demand
      - metric_name: total_energy
        expression: SUM(consumption_power * 0.25)
        description: Total energy in kWh
      - metric_name: self_sufficiency
        expression: SUM(solar_power) / NULLIF(SUM(consumption_power), 0) * 100
        description: Self-sufficiency percentage

  - table_name: energy_hourly
    schema: public
    columns:
      - column_name: hour
        type: TIMESTAMP
        is_dttm: true
      - column_name: avg_solar
        type: FLOAT
      - column_name: total_solar
        type: FLOAT
      - column_name: total_consumption
        type: FLOAT

  - table_name: energy_daily
    schema: public
    columns:
      - column_name: day
        type: TIMESTAMP
        is_dttm: true
      - column_name: peak_solar
        type: FLOAT
      - column_name: daily_solar_yield
        type: FLOAT
      - column_name: daily_consumption
        type: FLOAT
      - column_name: daily_import
        type: FLOAT
      - column_name: daily_export
        type: FLOAT
EOF
    
    log_info "Superset dataset configuration created at /tmp/superset-openems-dataset.yaml"
    return 0
}

# -------------------------------
# Integration Tests
# -------------------------------

test_superset_connectivity() {
    log_info "Testing Apache Superset connectivity"
    
    if curl -sf "${SUPERSET_URL}/health" &>/dev/null; then
        log_success "Superset is reachable at ${SUPERSET_URL}"
        return 0
    else
        log_warn "Superset is not reachable. Please ensure it's running."
        echo "Start Superset with: vrooli resource apache-superset manage start"
        return 1
    fi
}

test_questdb_connectivity() {
    log_info "Testing QuestDB connectivity"
    
    if curl -sf "${QUESTDB_URL}/exec?query=SELECT%20now();" &>/dev/null; then
        log_success "QuestDB is reachable at ${QUESTDB_URL}"
        return 0
    else
        log_warn "QuestDB is not reachable. Please ensure it's running."
        echo "Start QuestDB with: vrooli resource questdb manage start"
        return 1
    fi
}

# -------------------------------
# Deployment Instructions
# -------------------------------

show_deployment_instructions() {
    echo "═══════════════════════════════════════════════════════════════"
    echo "           Superset Dashboard Deployment Instructions          "
    echo "═══════════════════════════════════════════════════════════════"
    echo ""
    echo "1. ENSURE DEPENDENCIES ARE RUNNING:"
    echo "   vrooli resource questdb manage start"
    echo "   vrooli resource apache-superset manage start"
    echo "   vrooli resource openems manage start"
    echo ""
    echo "2. CONFIGURE QUESTDB CONNECTION IN SUPERSET:"
    echo "   - Open Superset: ${SUPERSET_URL}"
    echo "   - Go to: Data → Databases → + Database"
    echo "   - Select: PostgreSQL"
    echo "   - Connection string: postgresql://admin:quest@localhost:8812/qdb"
    echo "   - Display name: QuestDB OpenEMS"
    echo "   - Test & Save connection"
    echo ""
    echo "3. CREATE QUESTDB VIEWS:"
    echo "   Execute the SQL file in QuestDB console (${QUESTDB_URL}):"
    echo "   /tmp/questdb-openems-views.sql"
    echo ""
    echo "4. IMPORT DATASETS TO SUPERSET:"
    echo "   - Go to: Data → Datasets → + Dataset"
    echo "   - Database: QuestDB OpenEMS"
    echo "   - Schema: public"
    echo "   - Tables: openems_telemetry, energy_hourly, energy_daily"
    echo ""
    echo "5. IMPORT DASHBOARD TEMPLATES:"
    echo "   - Go to: Dashboards → Import"
    echo "   - Select templates from /tmp/:"
    echo "     • energy-overview-dashboard.json"
    echo "     • solar-analytics-dashboard.json"
    echo "     • battery-management-dashboard.json"
    echo "     • grid-interaction-dashboard.json"
    echo ""
    echo "6. CUSTOMIZE DASHBOARDS:"
    echo "   - Edit charts to match your data columns"
    echo "   - Adjust time ranges and filters"
    echo "   - Set auto-refresh intervals"
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
}

# -------------------------------
# Main execution
# -------------------------------

main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        create-dashboards)
            create_energy_overview_dashboard
            create_solar_analytics_dashboard
            create_battery_management_dashboard
            create_grid_interaction_dashboard
            create_questdb_views
            create_superset_dataset_config
            echo ""
            log_success "All dashboard templates created in /tmp/"
            show_deployment_instructions
            ;;
        test)
            test_superset_connectivity
            test_questdb_connectivity
            ;;
        deploy-instructions)
            show_deployment_instructions
            ;;
        help|*)
            echo "OpenEMS Superset Dashboard Integration"
            echo ""
            echo "Commands:"
            echo "  create-dashboards    - Create all dashboard templates"
            echo "  test                - Test connectivity to Superset and QuestDB"
            echo "  deploy-instructions - Show deployment instructions"
            echo ""
            echo "Usage:"
            echo "  ${0} create-dashboards"
            echo "  ${0} test"
            echo "  ${0} deploy-instructions"
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi