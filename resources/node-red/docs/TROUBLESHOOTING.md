# Node-RED Troubleshooting Guide

This document provides solutions for common Node-RED issues and problems.

## 🔧 Troubleshooting

### Common Issues

**Node-RED not accessible**
```bash
# Check if container is running
./manage.sh --action status

# Check logs for errors
./manage.sh --action logs --lines 50

# Verify port is accessible
curl http://localhost:1880
```

**Flows not loading**
```bash
# Check flows file
cat data/flows.json

# Validate JSON format
jq . data/flows.json

# Restart with safe mode
docker exec node-red-vrooli touch /data/.config.json
./manage.sh --action restart
```

**Permission errors**
```bash
# Fix data directory permissions
sudo chown -R 1000:1000 data/

# Restart container
./manage.sh --action restart
```

**Memory issues**
```bash
# Check resource usage
./manage.sh --action metrics

# Monitor real-time usage
./manage.sh --action monitor
```

## 🔍 Diagnostic Commands

### Container Status
```bash
# Check if container exists
docker ps -a | grep node-red

# View container logs
docker logs node-red-vrooli

# Inspect container configuration  
docker inspect node-red-vrooli
```

### Network Issues
```bash
# Check port binding
docker port node-red-vrooli

# Test network connectivity
curl -I http://localhost:1880

# Verify Docker network
docker network ls | grep node-red
```

### Data Directory Issues
```bash
# Check data directory exists
ls -la data/

# Verify permissions
stat data/

# Check disk space
df -h data/
```

### Performance Issues
```bash
# Monitor resource usage
docker stats node-red-vrooli

# Check system load
top -p $(docker inspect -f '{{.State.Pid}}' node-red-vrooli)

# Review container logs for errors
docker logs --tail 100 node-red-vrooli | grep -i error
```

## 🚨 Emergency Recovery

### Container Won't Start
```bash
# Remove and recreate container
./manage.sh --action stop
docker rm node-red-vrooli
./manage.sh --action install --force yes
```

### Corrupted Flows
```bash
# Restore from backup
./manage.sh --action restore --backup-path <backup-file>

# Or create empty flows file
echo '[]' > data/flows.json
./manage.sh --action restart
```

### Complete Reset
```bash
# Stop and remove everything
./manage.sh --action stop
docker rm node-red-vrooli
sudo rm -rf data/

# Reinstall fresh
./manage.sh --action install --force yes
```

## 📞 Getting Help

### Log Analysis
```bash
# Export logs for analysis
./manage.sh --action logs --lines 1000 > node-red-debug.log

# Check for specific error patterns
grep -i "error\|fatal\|exception" node-red-debug.log
```

### System Information
```bash
# Get system info for support requests
./manage.sh --action info
./manage.sh --action health
docker version
docker-compose version
```

### Community Resources
- [Node-RED Community Forum](https://discourse.nodered.org/)
- [Node-RED Slack](https://nodered.org/slack/)
- [GitHub Issues](https://github.com/node-red/node-red/issues)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/node-red)