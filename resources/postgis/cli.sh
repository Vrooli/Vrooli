#!/usr/bin/env bash
################################################################################
# PostGIS Resource CLI - v2.0 Universal Contract Compliant
# 
# Spatial database extension for PostgreSQL with GIS capabilities
#
# Usage:
#   resource-postgis <command> [options]
#   resource-postgis <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    POSTGIS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${POSTGIS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
POSTGIS_CLI_DIR="${APP_ROOT}/resources/postgis"

# Source standard variables
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities
# shellcheck disable=SC2154  # var_LOG_FILE is set by var.sh
source "${var_LOG_FILE}"
# shellcheck disable=SC2154  # var_RESOURCES_COMMON_FILE is set by var.sh
source "${var_RESOURCES_COMMON_FILE}"

# Source v2.0 CLI Command Framework
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source resource configuration
source "${POSTGIS_CLI_DIR}/config/defaults.sh"

# Source resource libraries (only what exists)
for lib in core common install status test inject performance health integration visualization geocoding spatial_analysis cleanup; do
    lib_file="${POSTGIS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "postgis" "Spatial database extension for PostgreSQL" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
# shellcheck disable=SC2034  # CLI_COMMAND_HANDLERS is used by CLI framework
CLI_COMMAND_HANDLERS["manage::install"]="postgis::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="postgis::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="postgis::docker::start"
CLI_COMMAND_HANDLERS["manage::stop"]="postgis::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="postgis::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="postgis::status::check"
CLI_COMMAND_HANDLERS["test::unit"]="postgis::test::unit"
CLI_COMMAND_HANDLERS["test::integration"]="postgis::test::integration"
CLI_COMMAND_HANDLERS["test::all"]="postgis::test::all"
CLI_COMMAND_HANDLERS["test::extended"]="postgis::test::extended"
CLI_COMMAND_HANDLERS["test::geocoding"]="postgis::test::geocoding"
CLI_COMMAND_HANDLERS["test::spatial"]="postgis::test::spatial"
CLI_COMMAND_HANDLERS["test::visualization"]="postgis::test::visualization"

# Content handlers for spatial data management
# shellcheck disable=SC2034  # CLI_COMMAND_HANDLERS is used by CLI framework
CLI_COMMAND_HANDLERS["content::add"]="postgis::content::add"
# shellcheck disable=SC2034  # CLI_COMMAND_HANDLERS is used by CLI framework
CLI_COMMAND_HANDLERS["content::list"]="postgis::content::list"
# shellcheck disable=SC2034  # CLI_COMMAND_HANDLERS is used by CLI framework
CLI_COMMAND_HANDLERS["content::get"]="postgis::content::get"
# shellcheck disable=SC2034  # CLI_COMMAND_HANDLERS is used by CLI framework
CLI_COMMAND_HANDLERS["content::remove"]="postgis::content::remove"
# shellcheck disable=SC2034  # CLI_COMMAND_HANDLERS is used by CLI framework
CLI_COMMAND_HANDLERS["content::execute"]="postgis::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed PostGIS status" "postgis::status"
cli::register_command "logs" "Show PostGIS container logs" "postgis::docker::logs"
cli::register_command "credentials" "Display integration credentials" "postgis::credentials"

# ==============================================================================
# OPTIONAL RESOURCE-SPECIFIC COMMANDS FOR SPATIAL DATA
# ==============================================================================
# Add spatial-specific content subcommands
cli::register_subcommand "content" "import-shapefile" "Import shapefile to PostGIS" "postgis_import_shapefile" "modifies-system"
cli::register_subcommand "content" "import-geojson" "Import GeoJSON to PostGIS" "postgis_import_geojson" "modifies-system"
cli::register_subcommand "content" "import-kml" "Import KML/KMZ to PostGIS" "postgis_import_kml" "modifies-system"
cli::register_subcommand "content" "import-gis" "Import any supported GIS format" "postgis_import_gis" "modifies-system"
cli::register_subcommand "content" "export-shapefile" "Export PostGIS table to shapefile" "postgis_export_shapefile" "modifies-system"
cli::register_subcommand "content" "enable-database" "Enable PostGIS in specific database" "postgis_enable_database" "modifies-system"
cli::register_subcommand "content" "disable-database" "Disable PostGIS in specific database" "postgis_disable_database" "modifies-system"

# Add spatial query examples command
cli::register_command "examples" "Show example spatial queries" "postgis_show_examples"

# Register optional test subcommands (P2 features)
cli::register_subcommand "test" "extended" "Run core + P2 feature tests" "postgis::test::extended"
cli::register_subcommand "test" "geocoding" "Run geocoding tests (P2)" "postgis::test::geocoding"
cli::register_subcommand "test" "spatial" "Run spatial analysis tests (P2)" "postgis::test::spatial"
cli::register_subcommand "test" "visualization" "Run visualization tests (P2)" "postgis::test::visualization"

# ==============================================================================
# PERFORMANCE OPTIMIZATION COMMANDS
# ==============================================================================
# Register performance command group
cli::register_command_group "performance" "Performance optimization and tuning"

# Performance subcommands
cli::register_subcommand "performance" "analyze-indexes" "Analyze spatial indexes" "postgis::performance::analyze_indexes"
cli::register_subcommand "performance" "analyze-query" "Analyze query performance" "postgis::performance::analyze_query"
cli::register_subcommand "performance" "tune-config" "Show configuration tuning recommendations" "postgis::performance::tune_config"
cli::register_subcommand "performance" "create-index" "Create optimized spatial index" "postgis::performance::create_spatial_index" "modifies-system"
cli::register_subcommand "performance" "vacuum" "Vacuum and analyze spatial tables" "postgis::performance::vacuum_analyze" "modifies-system"
cli::register_subcommand "performance" "stats" "Show performance statistics" "postgis::performance::show_stats"

# ==============================================================================
# CROSS-RESOURCE INTEGRATION COMMANDS
# ==============================================================================
# Register integration command group
cli::register_command_group "integration" "Cross-resource integration"

# Integration subcommands
cli::register_subcommand "integration" "status" "Show integration status" "postgis::integration::status"
cli::register_subcommand "integration" "export-n8n" "Export data for n8n workflows" "postgis::integration::export_for_n8n"
cli::register_subcommand "integration" "setup-ollama" "Setup Ollama embeddings table" "postgis::integration::create_ollama_embeddings_table" "modifies-system"
cli::register_subcommand "integration" "setup-questdb" "Setup QuestDB time-series sync" "postgis::integration::setup_questdb_sync" "modifies-system"
cli::register_subcommand "integration" "setup-redis" "Setup Redis cache tables" "postgis::integration::create_redis_cache_tables" "modifies-system"

# Visualization subcommands (P2 feature)
cli::register_command "visualization" "Generate map visualizations" "postgis::visualization::main"
cli::register_subcommand "visualization" "geojson" "Generate GeoJSON from query" "postgis::visualization::generate_geojson"
cli::register_subcommand "visualization" "heatmap" "Generate heat map" "postgis::visualization::generate_heatmap"
cli::register_subcommand "visualization" "choropleth" "Generate choropleth map" "postgis::visualization::generate_choropleth"
cli::register_subcommand "visualization" "tiles" "Generate map tiles" "postgis::visualization::generate_tiles"
cli::register_subcommand "visualization" "viewer" "Generate HTML viewer" "postgis::visualization::generate_viewer"

# Geocoding subcommands (P2 feature)
cli::register_command "geocoding" "Address geocoding services" "postgis::geocoding::main"
cli::register_subcommand "geocoding" "init" "Initialize geocoding tables" "postgis::geocoding::init" "modifies-system"
cli::register_subcommand "geocoding" "geocode" "Convert address to coordinates" "postgis::geocoding::geocode"
cli::register_subcommand "geocoding" "reverse" "Convert coordinates to address" "postgis::geocoding::reverse"
cli::register_subcommand "geocoding" "batch" "Batch geocode addresses" "postgis::geocoding::batch"
cli::register_subcommand "geocoding" "import" "Import places data" "postgis::geocoding::import_places" "modifies-system"
cli::register_subcommand "geocoding" "stats" "Show geocoding statistics" "postgis::geocoding::stats"

# Spatial analysis subcommands (P2 feature)
cli::register_command "spatial" "Advanced spatial analysis" "postgis::spatial::main"
cli::register_subcommand "spatial" "init-routing" "Initialize routing tables" "postgis::spatial::init_routing" "modifies-system"
cli::register_subcommand "spatial" "shortest-path" "Find shortest path" "postgis::spatial::shortest_path"
cli::register_subcommand "spatial" "proximity" "Find nearby features" "postgis::spatial::proximity"
cli::register_subcommand "spatial" "service-area" "Calculate service area" "postgis::spatial::service_area"
cli::register_subcommand "spatial" "watershed" "Perform watershed analysis" "postgis::spatial::watershed"
cli::register_subcommand "spatial" "viewshed" "Calculate viewshed" "postgis::spatial::viewshed"
cli::register_subcommand "spatial" "cluster" "Perform spatial clustering" "postgis::spatial::cluster"
cli::register_subcommand "spatial" "statistics" "Calculate spatial statistics" "postgis::spatial::statistics"

# Cleanup commands for test data management
cli::register_command "cleanup" "Cleanup test data and caches" "postgis::cleanup::all"
cli::register_subcommand "cleanup" "test-tables" "Remove test tables" "postgis::cleanup::test_tables"
cli::register_subcommand "cleanup" "test-databases" "Remove test databases" "postgis::cleanup::test_databases"
cli::register_subcommand "cleanup" "geocoding-cache" "Clear geocoding cache" "postgis::cleanup::geocoding_cache"
cli::register_subcommand "cleanup" "all" "Full cleanup of all test data" "postgis::cleanup::all"

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi