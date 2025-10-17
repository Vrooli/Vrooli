#!/usr/bin/env bash
# Open MCT Telemetry Demo Script

# Push sample telemetry data to Open MCT
echo "Sending demo telemetry data to Open MCT..."

# Send temperature sensor data
curl -X POST http://localhost:8099/api/telemetry/temperature_sensor/data \
  -H "Content-Type: application/json" \
  -d '{
    "timestamp": '$(date +%s000)',
    "value": '$((20 + RANDOM % 10))',
    "data": {
      "sensor_id": "TEMP-001",
      "location": "Server Room",
      "unit": "celsius"
    }
  }'

echo "Temperature data sent"

# Send pressure sensor data
curl -X POST http://localhost:8099/api/telemetry/pressure_sensor/data \
  -H "Content-Type: application/json" \
  -d '{
    "timestamp": '$(date +%s000)',
    "value": '$((1000 + RANDOM % 50))',
    "data": {
      "sensor_id": "PRESS-001",
      "location": "Chamber A",
      "unit": "mbar"
    }
  }'

echo "Pressure data sent"

# Send system performance metrics
curl -X POST http://localhost:8099/api/telemetry/system_performance/data \
  -H "Content-Type: application/json" \
  -d '{
    "timestamp": '$(date +%s000)',
    "value": '$((RANDOM % 100))',
    "data": {
      "cpu_usage": '$((RANDOM % 100))',
      "memory_usage": '$((RANDOM % 100))',
      "disk_usage": '$((RANDOM % 100))',
      "network_throughput": '$((RANDOM % 1000))'
    }
  }'

echo "System metrics sent"

echo ""
echo "Demo telemetry data sent successfully!"
echo "View the data at: http://localhost:8099"