# QuestDB Examples

This directory contains practical examples demonstrating QuestDB integration with Vrooli's resource ecosystem.

## 📁 Directory Structure

- **basic/** - Simple examples for getting started
- **ai-metrics/** - AI model performance tracking examples
- **resource-monitoring/** - Resource health and monitoring patterns
- **integration/** - Integration with other Vrooli resources

## 🚀 Quick Start

Run the basic example to see QuestDB in action:
```bash
cd basic
./quick-start.sh
```

## 📊 Example Categories

### Basic Usage
- Connecting to QuestDB
- Inserting data via different protocols
- Querying time-series data
- Using the web console

### AI Metrics Tracking
- Recording model inference times
- Tracking token usage and costs
- Analyzing model performance trends
- Identifying optimization opportunities

### Resource Monitoring
- Health check automation
- Performance baselines
- Alert generation
- Capacity planning

### Integration Examples
- **n8n workflows** - Automated monitoring and alerting
- **Node-RED flows** - Real-time dashboards
- **Python scripts** - Data analysis and reporting
- **Shell scripts** - Metric collection automation

## 💡 Common Patterns

### High-Speed Ingestion
```bash
# Use InfluxDB Line Protocol for fastest ingestion
echo "metric,tag=value field=123 $(date +%s)000000000" | nc localhost 9003
```

### Time-Series Queries
```sql
-- Aggregate by time intervals
SELECT timestamp, AVG(value) 
FROM metrics 
WHERE timestamp > now() - 1d 
SAMPLE BY 1h;
```

### Performance Monitoring
```sql
-- Find performance outliers
SELECT * FROM ai_inference 
WHERE response_time_ms > (
  SELECT AVG(response_time_ms) + 2 * STDDEV(response_time_ms) 
  FROM ai_inference
);
```

## 🔗 Integration Tips

### With n8n
- Use HTTP Request nodes for queries
- Schedule regular metric collection
- Create alert workflows based on thresholds

### With Node-RED
- Use inject nodes for periodic queries
- Create real-time dashboards
- Stream metrics via WebSocket

### With Python/Scripts
- Use requests library for REST API
- PostgreSQL libraries work via port 8812
- Batch insert for better performance

## 📚 Further Reading

- [QuestDB Documentation](https://questdb.io/docs/)
- [Time-Series Best Practices](https://questdb.io/docs/concept/designated-timestamp/)
- [SQL Reference](https://questdb.io/docs/reference/sql/)