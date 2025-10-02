#!/usr/bin/env bash
# OpenTripPlanner Core Functions

set -euo pipefail

# Resource configuration
readonly OPENTRIPPLANNER_IMAGE="opentripplanner/opentripplanner:latest"
readonly OPENTRIPPLANNER_CONTAINER="vrooli-opentripplanner"
readonly OPENTRIPPLANNER_NETWORK="vrooli-network"

# Get configuration from environment or defaults
export OTP_PORT="${OTP_PORT:-8080}"
export OTP_HEAP_SIZE="${OTP_HEAP_SIZE:-2G}"
export OTP_BUILD_TIMEOUT="${OTP_BUILD_TIMEOUT:-300}"
export OTP_DATA_DIR="${OTP_DATA_DIR:-${HOME}/.vrooli/opentripplanner/data}"
export OTP_CACHE_DIR="${OTP_CACHE_DIR:-${HOME}/.vrooli/opentripplanner/cache}"

# Lifecycle Management Functions

opentripplanner::install() {
    local force="${1:-false}"
    
    if [[ "$force" != "--force" ]] && opentripplanner::is_installed; then
        echo "OpenTripPlanner is already installed"
        return 2
    fi
    
    echo "Installing OpenTripPlanner..."
    
    # Create directories
    mkdir -p "${OTP_DATA_DIR}" "${OTP_CACHE_DIR}"
    
    # Pull Docker image
    docker pull "${OPENTRIPPLANNER_IMAGE}" || {
        echo "Failed to pull OpenTripPlanner image"
        return 1
    }
    
    # Create Docker network if it doesn't exist
    docker network create "${OPENTRIPPLANNER_NETWORK}" &>/dev/null || true
    
    # Download sample data (Portland)
    opentripplanner::download_sample_data || {
        echo "Warning: Failed to download sample data"
    }
    
    echo "OpenTripPlanner installed successfully"
    return 0
}

opentripplanner::uninstall() {
    local keep_data="${1:-false}"
    
    echo "Uninstalling OpenTripPlanner..."
    
    # Stop service if running
    opentripplanner::stop --force &>/dev/null || true
    
    # Remove container
    docker rm -f "${OPENTRIPPLANNER_CONTAINER}" &>/dev/null || true
    
    # Remove image
    docker rmi "${OPENTRIPPLANNER_IMAGE}" &>/dev/null || true
    
    # Remove data if requested
    if [[ "$keep_data" != "--keep-data" ]]; then
        rm -rf "${OTP_DATA_DIR}" "${OTP_CACHE_DIR}"
    fi
    
    echo "OpenTripPlanner uninstalled successfully"
    return 0
}

opentripplanner::start() {
    local wait_flag="${1:-}"
    local timeout="${2:-60}"
    
    if opentripplanner::is_running; then
        echo "OpenTripPlanner is already running"
        return 2
    fi
    
    echo "Starting OpenTripPlanner..."
    
    # Ensure data directory exists
    mkdir -p "${OTP_DATA_DIR}" "${OTP_CACHE_DIR}"
    
    # Start container - OTP 2.x uses different command structure
    # We need to provide a router config to serve properly
    
    # Create OTP v2.x router config if it doesn't exist
    if [[ ! -f "${OTP_DATA_DIR}/router-config.json" ]]; then
        cat > "${OTP_DATA_DIR}/router-config.json" <<'EOF'
{
  "routingDefaults": {
    "numItineraries": 3
  },
  "updaters": [],
  "transit": {
    "dynamicSearchWindow": {
      "minTransitTimeCoefficient": 0.5,
      "minTimeMinutes": 45,
      "maxLengthMinutes": 90
    }
  }
}
EOF
        echo "Created OTP v2.x router configuration"
    fi
    
    # Check if graph exists
    local graph_exists=false
    if [[ -f "${OTP_DATA_DIR}/graph.obj" ]] || [[ -f "${OTP_CACHE_DIR}/graph.obj" ]]; then
        graph_exists=true
        echo "Found existing graph, will load it..."
    else
        echo "No graph found, starting without graph (API only)..."
    fi
    
    # OTP 2.x container with entrypoint handling directory
    # The entrypoint adds /var/opentripplanner automatically
    # Check if graph exists to determine the right mode
    local otp_args=""
    if [[ -f "${OTP_DATA_DIR}/graph.obj" ]]; then
        # OTP 2.x uses --load for loading an existing graph
        otp_args="--load"
        echo "Using existing graph..."
    else
        # Without a graph, we'll create one first if there's data
        if ls "${OTP_DATA_DIR}"/*.gtfs.zip &>/dev/null 2>&1 || ls "${OTP_DATA_DIR}"/*.pbf &>/dev/null 2>&1; then
            echo "Data files found, will build and serve..."
            otp_args="--build --save"
        else
            # For now, let's just serve without a graph (will return empty results)
            echo "No data files found. Starting API server only..."
            otp_args="--load"
        fi
    fi

    docker run -d \
        --name "${OPENTRIPPLANNER_CONTAINER}" \
        --network "${OPENTRIPPLANNER_NETWORK}" \
        -p "${OTP_PORT}:8080" \
        -v "${OTP_DATA_DIR}:/var/opentripplanner" \
        -v "${OTP_CACHE_DIR}:/var/cache/otp" \
        -e JAVA_OPTS="-Xmx${OTP_HEAP_SIZE}" \
        "${OPENTRIPPLANNER_IMAGE}" \
        $otp_args || {
        echo "Failed to start OpenTripPlanner"
        return 1
    }
    
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "Waiting for OpenTripPlanner to be ready..."
        if ! opentripplanner::wait_for_ready "$timeout"; then
            echo "OpenTripPlanner failed to become ready within ${timeout} seconds"
            return 1
        fi
    fi
    
    echo "OpenTripPlanner started successfully on port ${OTP_PORT}"
    return 0
}

opentripplanner::stop() {
    local force="${1:-}"
    local timeout="${2:-30}"
    
    if ! opentripplanner::is_running; then
        echo "OpenTripPlanner is not running"
        return 2
    fi
    
    echo "Stopping OpenTripPlanner..."
    
    if [[ "$force" == "--force" ]]; then
        docker kill "${OPENTRIPPLANNER_CONTAINER}" &>/dev/null || true
    else
        docker stop -t "$timeout" "${OPENTRIPPLANNER_CONTAINER}" &>/dev/null || true
    fi
    
    docker rm "${OPENTRIPPLANNER_CONTAINER}" &>/dev/null || true
    
    echo "OpenTripPlanner stopped successfully"
    return 0
}

opentripplanner::restart() {
    local force="${1:-}"
    
    opentripplanner::stop "$force" || true
    opentripplanner::start --wait
}

# Status Functions

opentripplanner::status() {
    local format="${1:-text}"
    local verbose="${2:-}"
    
    local running="false"
    local healthy="false"
    local port="${OTP_PORT}"
    local container_status="stopped"
    
    if opentripplanner::is_running; then
        running="true"
        container_status="running"
        
        if opentripplanner::is_healthy; then
            healthy="true"
        fi
    fi
    
    if [[ "$format" == "--json" || "$format" == "json" ]]; then
        cat <<EOF
{
  "service": "opentripplanner",
  "running": ${running},
  "healthy": ${healthy},
  "port": ${port},
  "container_status": "${container_status}",
  "data_dir": "${OTP_DATA_DIR}",
  "cache_dir": "${OTP_CACHE_DIR}"
}
EOF
    else
        echo "OpenTripPlanner Status:"
        echo "  Running: ${running}"
        echo "  Healthy: ${healthy}"
        echo "  Port: ${port}"
        echo "  Container: ${container_status}"
        
        if [[ "$verbose" == "--verbose" ]]; then
            echo "  Data Directory: ${OTP_DATA_DIR}"
            echo "  Cache Directory: ${OTP_CACHE_DIR}"
            echo "  Heap Size: ${OTP_HEAP_SIZE}"
        fi
    fi
    
    [[ "$healthy" == "true" ]] && return 0 || return 1
}

opentripplanner::logs() {
    local tail="${1:-50}"
    local follow="${2:-}"
    
    if [[ "$follow" == "--follow" ]]; then
        docker logs -f --tail "$tail" "${OPENTRIPPLANNER_CONTAINER}"
    else
        docker logs --tail "$tail" "${OPENTRIPPLANNER_CONTAINER}"
    fi
}

# Content Management Functions

opentripplanner::content::add() {
    local file=""
    local url=""
    local type=""
    local name=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file) file="$2"; shift 2 ;;
            --url) url="$2"; shift 2 ;;
            --type) type="$2"; shift 2 ;;
            --name) name="$2"; shift 2 ;;
            *) shift ;;
        esac
    done

    if [[ -z "$name" || -z "$type" ]] || [[ -z "$file" && -z "$url" ]]; then
        echo "Usage: content add --type <gtfs|osm|gtfs-rt> --name <identifier> [--file <path>|--url <url>]"
        return 1
    fi

    local dest_dir="${OTP_DATA_DIR}"

    case "$type" in
        gtfs)
            if [[ -n "$file" ]]; then
                cp "$file" "${dest_dir}/${name}.gtfs.zip"
            else
                curl -L -o "${dest_dir}/${name}.gtfs.zip" "$url"
            fi
            echo "Added GTFS data: ${name}"
            ;;
        osm)
            if [[ -n "$file" ]]; then
                cp "$file" "${dest_dir}/${name}.osm.pbf"
            else
                curl -L -o "${dest_dir}/${name}.osm.pbf" "$url"
            fi
            echo "Added OSM data: ${name}"
            ;;
        gtfs-rt)
            # Store GTFS-RT feed URL in router config
            opentripplanner::add_gtfs_rt_feed "$name" "${url:-$file}"
            echo "Added GTFS-RT feed: ${name}"
            ;;
        *)
            echo "Unknown type: $type (use gtfs, osm, or gtfs-rt)"
            return 1
            ;;
    esac

    return 0
}

opentripplanner::content::list() {
    local format="${1:-text}"

    echo "Available data files:"
    echo ""
    echo "GTFS Files:"
    ls -la "${OTP_DATA_DIR}"/*.gtfs.zip 2>/dev/null || echo "  No GTFS files"
    echo ""
    echo "OSM Files:"
    ls -la "${OTP_DATA_DIR}"/*.osm.pbf 2>/dev/null || echo "  No OSM files"
    echo ""
    echo "Graph Cache:"
    ls -la "${OTP_DATA_DIR}"/graph*.obj 2>/dev/null || echo "  No cached graphs"
    echo ""
    echo "GTFS-RT Feeds:"
    if [[ -f "${OTP_DATA_DIR}/router-config.json" ]]; then
        local feeds=$(jq -r '.updaters[]? | select(.type == "real-time-alerts") | "  - \(.feedId): \(.url)"' "${OTP_DATA_DIR}/router-config.json" 2>/dev/null)
        if [[ -n "$feeds" ]]; then
            echo "$feeds"
        else
            echo "  No GTFS-RT feeds configured"
        fi
    else
        echo "  No router config found"
    fi
}

opentripplanner::content::get() {
    local name=""
    local output=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) name="$2"; shift 2 ;;
            --output) output="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Usage: content get --name <identifier> [--output <path>]"
        return 1
    fi
    
    local source_file=""
    
    # Try to find the file
    for ext in "gtfs.zip" "osm.pbf"; do
        if [[ -f "${OTP_DATA_DIR}/${name}.${ext}" ]]; then
            source_file="${OTP_DATA_DIR}/${name}.${ext}"
            break
        fi
    done
    
    if [[ -z "$source_file" ]]; then
        echo "Content not found: ${name}"
        return 2
    fi
    
    if [[ -n "$output" ]]; then
        cp "$source_file" "$output"
        echo "Retrieved content to: $output"
    else
        echo "Content found at: $source_file"
    fi
    
    return 0
}

opentripplanner::content::remove() {
    local name=""
    local force=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) name="$2"; shift 2 ;;
            --force) force="true"; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Usage: content remove --name <identifier> [--force]"
        return 1
    fi
    
    local found=false
    
    # Remove matching files
    for ext in "gtfs.zip" "osm.pbf"; do
        if [[ -f "${OTP_DATA_DIR}/${name}.${ext}" ]]; then
            rm -f "${OTP_DATA_DIR}/${name}.${ext}"
            echo "Removed: ${name}.${ext}"
            found=true
        fi
    done
    
    if [[ "$found" == "false" ]]; then
        echo "Content not found: ${name}"
        return 2
    fi
    
    return 0
}

opentripplanner::plan_trip() {
    local from_lat=""
    local from_lon=""
    local to_lat=""
    local to_lon=""
    local modes="bus,rail,tram"
    local output_format="summary"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --from-lat) from_lat="$2"; shift 2 ;;
            --from-lon) from_lon="$2"; shift 2 ;;
            --to-lat) to_lat="$2"; shift 2 ;;
            --to-lon) to_lon="$2"; shift 2 ;;
            --modes) modes="$2"; shift 2 ;;
            --format) output_format="$2"; shift 2 ;;
            *) shift ;;
        esac
    done

    # Validate required parameters
    if [[ -z "$from_lat" ]] || [[ -z "$from_lon" ]] || [[ -z "$to_lat" ]] || [[ -z "$to_lon" ]]; then
        echo "Error: Missing required coordinates"
        echo "Usage: --from-lat LAT --from-lon LON --to-lat LAT --to-lon LON"
        return 1
    fi

    # Check if OTP is running
    if ! opentripplanner::is_running; then
        echo "Error: OpenTripPlanner is not running"
        echo "Start it with: vrooli resource opentripplanner manage start"
        return 1
    fi

    # Build GraphQL query for Transmodel v3 API
    local transport_modes=""
    IFS=',' read -ra mode_array <<< "$modes"
    for mode in "${mode_array[@]}"; do
        # Map common mode names to OTP TransportMode enum values
        case "${mode,,}" in
            walk|foot) continue ;; # Walk is handled separately
            subway|underground) mode="metro" ;;
            streetcar) mode="tram" ;;
            ferry|boat) mode="water" ;;
            train) mode="rail" ;;
        esac
        transport_modes="${transport_modes}{ transportMode: ${mode} } "
    done

    # Create GraphQL query
    local query=$(cat <<EOF
{
  trip(
    from: {
      coordinates: {
        latitude: ${from_lat}
        longitude: ${from_lon}
      }
    }
    to: {
      coordinates: {
        latitude: ${to_lat}
        longitude: ${to_lon}
      }
    }
    modes: {
      transportModes: [ ${transport_modes} ]
    }
  ) {
    tripPatterns {
      startTime
      endTime
      duration
      walkDistance
      legs {
        mode
        distance
        duration
        line {
          publicCode
          name
        }
      }
    }
  }
}
EOF
)

    # Execute GraphQL query
    local response
    response=$(curl -sf -X POST "http://localhost:${OTP_PORT}/otp/transmodel/v3" \
        -H "Content-Type: application/json" \
        -d "{\"query\":$(echo "$query" | jq -Rs .)}" 2>/dev/null) || {
        echo "Error: Failed to query trip planner API"
        return 1
    }

    # Check for errors
    if echo "$response" | jq -e '.errors' &>/dev/null; then
        echo "Error planning trip:"
        echo "$response" | jq -r '.errors[0].message'
        return 1
    fi

    # Format output based on requested format
    case "$output_format" in
        json)
            echo "$response" | jq '.'
            ;;
        summary)
            # Parse and display trip summary
            local trip_count=$(echo "$response" | jq '.data.trip.tripPatterns | length')

            if [[ "$trip_count" -eq 0 ]] || [[ "$trip_count" == "null" ]]; then
                echo "No trips found for the given coordinates and modes"
                return 0
            fi

            echo "Found ${trip_count} trip option(s):"
            echo ""

            for ((i=0; i<trip_count; i++)); do
                local pattern=$(echo "$response" | jq ".data.trip.tripPatterns[$i]")
                local duration=$(echo "$pattern" | jq -r '.duration')
                local walk_distance=$(echo "$pattern" | jq -r '.walkDistance // 0')
                local leg_count=$(echo "$pattern" | jq '.legs | length')

                # Convert duration from seconds to minutes
                local duration_min=$((duration / 60))

                echo "Option $((i+1)):"
                echo "  Duration: ${duration_min} minutes"
                echo "  Walking distance: ${walk_distance} meters"
                echo "  Segments: ${leg_count}"

                # Show modes used
                local modes_used=$(echo "$pattern" | jq -r '.legs[].mode' | sort -u | tr '\n' ', ' | sed 's/,$//')
                echo "  Modes: ${modes_used}"
                echo ""
            done
            ;;
        *)
            echo "Unknown format: $output_format (use 'json' or 'summary')"
            return 1
            ;;
    esac

    return 0
}

opentripplanner::content::execute() {
    local action=""
    local name=""
    local store_results=""
    local analyze=""
    local remaining_args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --action) action="$2"; shift 2 ;;
            --name) name="$2"; shift 2 ;;
            --store-results) store_results="true"; shift ;;
            --analyze) analyze="true"; shift ;;
            *) remaining_args+=("$1"); shift ;;
        esac
    done

    case "$action" in
        build-graph)
            echo "Building routing graph from GTFS and OSM data..."

            # Stop the server if running to build the graph
            if opentripplanner::is_running; then
                echo "Stopping server to build graph..."
                opentripplanner::stop
            fi

            # Build graph using a temporary container
            # Entrypoint adds directory, we just provide --build --save flags
            docker run --rm \
                --network "${OPENTRIPPLANNER_NETWORK}" \
                -v "${OTP_DATA_DIR}:/var/opentripplanner" \
                -v "${OTP_CACHE_DIR}:/var/cache/otp" \
                -e JAVA_OPTS="-Xmx${OTP_HEAP_SIZE}" \
                "${OPENTRIPPLANNER_IMAGE}" \
                --build --save || {
                echo "Failed to build graph"
                return 1
            }

            echo "Graph built successfully"
            echo "You can now restart the server with: vrooli resource opentripplanner manage start"
            ;;

        analyze-routes)
            echo "Analyzing transit routes..."

            if [[ "$store_results" == "true" ]]; then
                # Check if PostGIS is available
                if vrooli resource status postgres --json 2>/dev/null | jq -e '.running' &>/dev/null; then
                    echo "PostGIS integration available - storing spatial data"

                    # Export transit stop locations to PostGIS
                    opentripplanner::export_to_postgis "stops"
                    echo "Transit stops exported to PostGIS"
                else
                    echo "Warning: PostgreSQL not running - skipping PostGIS storage"
                fi
            fi

            # Generate route statistics
            if [[ -f "${OTP_DATA_DIR}/graph.obj" ]]; then
                echo "Graph analysis completed"
                echo "  Transit stops: $(find "${OTP_DATA_DIR}" -name "*.gtfs.zip" -exec unzip -l {} \; 2>/dev/null | grep stops.txt | wc -l)"
                echo "  Route files: $(ls -1 "${OTP_DATA_DIR}"/*.gtfs.zip 2>/dev/null | wc -l)"
            else
                echo "No graph found - build graph first"
                return 1
            fi
            ;;

        export-stops)
            echo "Exporting transit stops..."
            opentripplanner::export_to_postgis "stops"
            ;;

        export-routes)
            echo "Exporting transit routes..."
            opentripplanner::export_to_postgis "routes"
            ;;

        plan-trip)
            opentripplanner::plan_trip "${remaining_args[@]}"
            ;;

        *)
            echo "Unknown action: $action"
            echo "Available actions: build-graph, analyze-routes, export-stops, export-routes, plan-trip"
            return 1
            ;;
    esac

    return 0
}

# Helper Functions

opentripplanner::is_installed() {
    docker image inspect "${OPENTRIPPLANNER_IMAGE}" &>/dev/null
}

opentripplanner::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${OPENTRIPPLANNER_CONTAINER}$"
}

opentripplanner::is_healthy() {
    # OTP 2.x health endpoint is at root or /otp/routers
    local health_url="http://localhost:${OTP_PORT}/otp/routers/default"
    # Try main API endpoint first, fallback to root
    timeout 5 curl -sf "${health_url}" &>/dev/null || \
    timeout 5 curl -sf "http://localhost:${OTP_PORT}/" &>/dev/null
}

opentripplanner::wait_for_ready() {
    local timeout="${1:-60}"
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if opentripplanner::is_healthy; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    echo ""
    
    return 1
}

opentripplanner::download_sample_data() {
    echo "Downloading sample GTFS and OSM data for Portland..."

    # Download Portland GTFS data
    local gtfs_url="https://developer.trimet.org/schedule/gtfs.zip"
    local gtfs_file="${OTP_DATA_DIR}/portland.gtfs.zip"

    if [[ ! -f "$gtfs_file" ]] || [[ ! -s "$gtfs_file" ]]; then
        echo "Downloading Portland GTFS data..."
        curl -L -o "$gtfs_file" "$gtfs_url" || {
            echo "Failed to download GTFS data"
            return 1
        }
        echo "Downloaded Portland GTFS data ($(du -h "$gtfs_file" | cut -f1))"
    else
        echo "Portland GTFS data already exists"
    fi

    # Download Portland OSM extract (smaller BBBike extract instead of whole Oregon)
    local osm_url="https://download.bbbike.org/osm/bbbike/Portland/Portland.osm.pbf"
    local osm_file="${OTP_DATA_DIR}/portland.osm.pbf"

    if [[ ! -f "$osm_file" ]] || [[ ! -s "$osm_file" ]]; then
        echo "Downloading Portland OSM data..."
        curl -L -o "$osm_file" "$osm_url" || {
            echo "Failed to download OSM data"
            return 1
        }
        echo "Downloaded Portland OSM data ($(du -h "$osm_file" | cut -f1))"
    else
        echo "Portland OSM data already exists"
    fi

    # Clean up any test/empty files
    find "${OTP_DATA_DIR}" -name "*.pbf" -size 0 -delete 2>/dev/null
    find "${OTP_DATA_DIR}" -name "*.zip" -size 0 -delete 2>/dev/null

    return 0
}

opentripplanner::add_gtfs_rt_feed() {
    local feed_id="$1"
    local feed_url="$2"

    # Read existing router config
    local config_file="${OTP_DATA_DIR}/router-config.json"

    # Update router config with GTFS-RT feed
    if [[ -f "$config_file" ]]; then
        # Use jq to add the GTFS-RT updater
        local updated_config=$(cat "$config_file" | jq --arg id "$feed_id" --arg url "$feed_url" '
            .updaters = (.updaters // []) + [{
                "type": "real-time-alerts",
                "sourceType": "gtfs-http",
                "feedId": $id,
                "url": $url,
                "frequencySec": 30
            }]
        ')

        echo "$updated_config" > "$config_file"
        echo "Added GTFS-RT feed to router configuration: $feed_id"
    else
        echo "Router config not found, creating new one with GTFS-RT feed"
        cat > "$config_file" <<EOF
{
  "routingDefaults": {
    "numItineraries": 3
  },
  "updaters": [
    {
      "type": "real-time-alerts",
      "sourceType": "gtfs-http",
      "feedId": "${feed_id}",
      "url": "${feed_url}",
      "frequencySec": 30
    }
  ],
  "transit": {
    "dynamicSearchWindow": {
      "minTransitTimeCoefficient": 0.5,
      "minTimeMinutes": 45,
      "maxLengthMinutes": 90
    }
  }
}
EOF
    fi

    return 0
}

opentripplanner::export_to_postgis() {
    local export_type="$1"

    # Check for PostgreSQL availability
    if ! vrooli resource status postgres --json 2>/dev/null | jq -e '.running' &>/dev/null; then
        echo "PostgreSQL is not running. Please start it first."
        return 1
    fi

    echo "Exporting $export_type to PostGIS..."

    # Create PostGIS schema for OTP data
    local sql_schema="CREATE SCHEMA IF NOT EXISTS opentripplanner;
    CREATE EXTENSION IF NOT EXISTS postgis;
    CREATE TABLE IF NOT EXISTS opentripplanner.transit_stops (
        stop_id VARCHAR(255) PRIMARY KEY,
        stop_name TEXT,
        stop_lat DOUBLE PRECISION,
        stop_lon DOUBLE PRECISION,
        geom GEOMETRY(Point, 4326)
    );
    CREATE INDEX IF NOT EXISTS idx_transit_stops_geom ON opentripplanner.transit_stops USING GIST(geom);"

    # Execute schema creation
    echo "$sql_schema" | docker exec -i vrooli-postgres psql -U postgres -d postgres 2>/dev/null || {
        echo "Failed to create PostGIS schema"
        return 1
    }

    case "$export_type" in
        stops)
            # Extract stops from GTFS files and insert into PostGIS
            for gtfs_file in "${OTP_DATA_DIR}"/*.gtfs.zip; do
                if [[ -f "$gtfs_file" ]]; then
                    # Extract stops.txt and convert to SQL inserts
                    unzip -p "$gtfs_file" stops.txt 2>/dev/null | tail -n +2 | while IFS=',' read -r stop_id stop_name stop_lat stop_lon rest; do
                        if [[ -n "$stop_id" && -n "$stop_lat" && -n "$stop_lon" ]]; then
                            echo "INSERT INTO opentripplanner.transit_stops (stop_id, stop_name, stop_lat, stop_lon, geom)
                                  VALUES ('$stop_id', '$stop_name', $stop_lat, $stop_lon, ST_SetSRID(ST_MakePoint($stop_lon, $stop_lat), 4326))
                                  ON CONFLICT (stop_id) DO UPDATE SET stop_name=EXCLUDED.stop_name, geom=EXCLUDED.geom;"
                        fi
                    done | docker exec -i vrooli-postgres psql -U postgres -d postgres 2>/dev/null
                fi
            done
            echo "Transit stops exported to PostGIS"
            ;;

        routes)
            echo "Route export to PostGIS is a placeholder for future implementation"
            ;;

        *)
            echo "Unknown export type: $export_type"
            return 1
            ;;
    esac

    return 0
}