#!/bin/bash

# Performance Metrics for Huginn
# Tracks detailed performance metrics per agent

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)}"

# Source required dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${SCRIPT_DIR}/common.sh"

# Get current performance metrics
get_performance_metrics() {
    local agent_id="${1:-}"
    
    local metrics_script=$(cat <<'RUBY'
require 'json'

agent_id = ENV['AGENT_ID']

# Collect overall metrics
total_agents = Agent.count
total_events = Event.count
total_jobs = DelayedJob.count
failed_jobs = DelayedJob.where('last_error IS NOT NULL').count

# Memory usage (approximate from database - MySQL version)
db_query = 'SELECT SUM(data_length + index_length) as size FROM information_schema.tables WHERE table_schema = DATABASE()'
db_result = ActiveRecord::Base.connection.execute(db_query).first rescue nil
db_size = db_result ? db_result[0].to_i : 0

# Agent-specific metrics if ID provided
agent_metrics = nil
if agent_id && !agent_id.empty?
  agent = Agent.find_by(id: agent_id.to_i)
  if agent
    agent_metrics = {
      name: agent.name,
      type: agent.type,
      events_created: agent.events.count,
      events_received: agent.received_events.count,
      last_check: agent.last_check_at,
      last_event: agent.events.order(created_at: :desc).first&.created_at,
      average_propagation_time: nil,
      error_count: agent.logs.where("level >= ?", 4).count,
      memory_usage: agent.memory&.to_json&.size || 0
    }
    
    # Calculate average event propagation time
    recent_events = agent.events.limit(100)
    if recent_events.any?
      prop_times = recent_events.map do |e|
        (e.created_at - agent.last_check_at).to_f rescue nil
      end.compact
      
      if prop_times.any?
        agent_metrics[:average_propagation_time] = (prop_times.sum / prop_times.size).round(3)
      end
    end
  end
end

# Performance over time (last 24 hours)
hour_ago = 1.hour.ago
day_ago = 24.hours.ago

hourly_events = Event.where(created_at: hour_ago..Time.now).count
daily_events = Event.where(created_at: day_ago..Time.now).count

# Agent performance rankings
agent_rankings = Agent.joins(:events)
  .where(events: { created_at: day_ago..Time.now })
  .group("agents.id")
  .order("COUNT(events.id) DESC")
  .limit(10)
  .pluck("agents.name", "agents.type", "COUNT(events.id)")
  .map { |name, type, count| { name: name, type: type, event_count: count } }

# Slowest agents (by last check duration)
slowest_agents = Agent.where("last_check_at IS NOT NULL")
  .order("last_check_at DESC")
  .limit(5)
  .map do |a|
    check_duration = (a.updated_at - a.last_check_at).to_f rescue nil
    {
      name: a.name,
      type: a.type,
      check_duration: check_duration&.round(3)
    }
  end.compact

metrics = {
  overview: {
    total_agents: total_agents,
    total_events: total_events,
    total_jobs: total_jobs,
    failed_jobs: failed_jobs,
    database_size_mb: (db_size / 1024.0 / 1024.0).round(2)
  },
  performance: {
    events_last_hour: hourly_events,
    events_last_24h: daily_events,
    events_per_hour: (daily_events / 24.0).round(2),
    top_performers: agent_rankings,
    slowest_agents: slowest_agents
  },
  agent: agent_metrics
}

puts JSON.pretty_generate(metrics)
RUBY
    )
    
    AGENT_ID="${agent_id}" docker exec huginn bash -c "cd /app && bundle exec rails runner -e production \"$metrics_script\""
}

# Create performance tracking agent
create_performance_monitor() {
    local name="${1:-Performance Monitor}"
    local check_schedule="${2:-*/15 * * * *}"  # Every 15 minutes by default
    
    log::info "Creating performance monitoring agent: ${name}"
    
    local create_script=$(cat <<'RUBY'
name = ENV['MONITOR_NAME']
schedule = ENV['CHECK_SCHEDULE']

# Create a performance monitoring agent
agent = Agent.build_for_type(
  'Agents::EventFormattingAgent',
  user: User.first,
  name: name,
  schedule: schedule,
  options: {
    'instructions' => {
      'timestamp' => '{{ "now" | date: "%Y-%m-%d %H:%M:%S" }}',
      'total_agents' => Agent.count.to_s,
      'total_events' => Event.count.to_s,
      'recent_events' => Event.where('created_at > ?', 1.hour.ago).count.to_s,
      'failed_jobs' => DelayedJob.where('last_error IS NOT NULL').count.to_s,
      'database_size_mb' => (ActiveRecord::Base.connection.execute("SELECT SUM(data_length + index_length) FROM information_schema.tables WHERE table_schema = DATABASE()").first[0].to_i / 1024.0 / 1024.0).round(2).to_s rescue "0"
    },
    'mode' => 'clean'
  }
)

if agent.save
  puts "SUCCESS: Created performance monitor '#{agent.name}' (ID: #{agent.id})"
else
  puts "ERROR: #{agent.errors.full_messages.join(', ')}"
  exit 1
end
RUBY
    )
    
    MONITOR_NAME="$name" \
    CHECK_SCHEDULE="$schedule" \
    docker exec huginn bash -c "cd /app && bundle exec rails runner -e production \"$create_script\""
}

# Show performance dashboard
performance_dashboard() {
    while true; do
        clear
        echo "════════════════════════════════════════════════════════════════"
        echo "                 HUGINN PERFORMANCE DASHBOARD"
        echo "════════════════════════════════════════════════════════════════"
        echo
        
        # Get current metrics
        local metrics
        metrics=$(get_performance_metrics 2>/dev/null)
        
        if [[ -n "$metrics" ]]; then
            # Parse and display metrics
            echo "$metrics" | jq -r '
              "📊 SYSTEM OVERVIEW",
              "├─ Total Agents: \(.overview.total_agents)",
              "├─ Total Events: \(.overview.total_events)",
              "├─ Queue Size: \(.overview.total_jobs)",
              "├─ Failed Jobs: \(.overview.failed_jobs)",
              "└─ Database Size: \(.overview.database_size_mb) MB",
              "",
              "⚡ PERFORMANCE METRICS",
              "├─ Events (Last Hour): \(.performance.events_last_hour)",
              "├─ Events (Last 24h): \(.performance.events_last_24h)",
              "└─ Average Rate: \(.performance.events_per_hour) events/hour",
              "",
              "🏆 TOP PERFORMERS (24h)",
              (.performance.top_performers[:5] | to_entries | map("├─ \(.value.name): \(.value.event_count) events") | join("\n")),
              "",
              "🐌 SLOWEST AGENTS",
              (.performance.slowest_agents[:5] | to_entries | map("├─ \(.value.name): \(.value.check_duration)s") | join("\n"))
            ' 2>/dev/null || echo "Error parsing metrics"
        else
            echo "Unable to fetch metrics. Is Huginn running?"
        fi
        
        echo
        echo "────────────────────────────────────────────────────────────────"
        echo "Press Ctrl+C to exit | Refreshing in 5 seconds..."
        sleep 5
    done
}

# Analyze agent performance
analyze_agent_performance() {
    local agent_id="$1"
    
    if [[ -z "$agent_id" ]]; then
        log::error "Agent ID required"
        return 1
    fi
    
    log::info "Analyzing performance for agent ${agent_id}..."
    
    local metrics
    metrics=$(get_performance_metrics "$agent_id")
    
    if [[ -n "$metrics" ]]; then
        echo "$metrics" | jq -r '
          if .agent then
            "📊 AGENT PERFORMANCE ANALYSIS",
            "════════════════════════════════════════",
            "",
            "Agent: \(.agent.name)",
            "Type: \(.agent.type)",
            "",
            "📈 EVENT STATISTICS",
            "├─ Events Created: \(.agent.events_created)",
            "├─ Events Received: \(.agent.events_received)",
            "├─ Last Check: \(.agent.last_check)",
            "├─ Last Event: \(.agent.last_event)",
            "└─ Avg Propagation: \(.agent.average_propagation_time)s",
            "",
            "⚠️ ERROR METRICS",
            "└─ Error Count: \(.agent.error_count)",
            "",
            "💾 RESOURCE USAGE",
            "└─ Memory Usage: \(.agent.memory_usage) bytes"
          else
            "Agent not found"
          end
        '
    else
        log::error "Failed to fetch agent metrics"
        return 1
    fi
}

# Export performance data
export_performance_data() {
    local output_file="${1:-huginn-performance-$(date +%Y%m%d-%H%M%S).json}"
    
    log::info "Exporting performance data to ${output_file}..."
    
    local metrics
    metrics=$(get_performance_metrics)
    
    if [[ -n "$metrics" ]]; then
        echo "$metrics" > "$output_file"
        log::success "Performance data exported to ${output_file}"
    else
        log::error "Failed to export performance data"
        return 1
    fi
}

# Main command handler
main() {
    local command="$1"
    shift
    
    case "$command" in
        dashboard)
            performance_dashboard
            ;;
        metrics)
            get_performance_metrics "$@" | jq .
            ;;
        analyze)
            analyze_agent_performance "$@"
            ;;
        export)
            export_performance_data "$@"
            ;;
        create-monitor)
            create_performance_monitor "$@"
            ;;
        *)
            log::error "Unknown command: $command"
            echo "Available commands: dashboard, metrics, analyze, export, create-monitor"
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi