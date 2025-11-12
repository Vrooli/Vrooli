# Troubleshooting Guide

This comprehensive guide helps diagnose and resolve common issues with the Qdrant Semantic Knowledge System.

## 🚨 Quick Diagnostic Commands

### Health Check Commands

```bash
# Check overall system health
resource-qdrant embeddings status

# Validate setup and configuration
resource-qdrant embeddings validate

# Test Qdrant connectivity
curl -s http://localhost:6333/ | jq

# Check collections
curl -s http://localhost:6333/collections | jq '.result[].name'

# Check if embeddings exist for current app
ls -la .vrooli/app-identity.json
```

### Quick Fixes

```bash
# Reset and refresh embeddings (safe operation)
resource-qdrant embeddings refresh --force

# Clear cache and restart
redis-cli FLUSHDB
vrooli develop --restart

# Rebuild app identity
rm .vrooli/app-identity.json
resource-qdrant embeddings init
```

## 🔧 Installation and Setup Issues

### Issue: Qdrant Resource Not Available

**Symptoms:**
```bash
$ resource-qdrant
Error: resource-qdrant command not found
```

**Diagnosis:**
```bash
# Check if Qdrant is enabled
cat .vrooli/service.json | jq '.dependencies.resources.qdrant'

# Check Qdrant resource installation
ls -la resources/qdrant/
```

**Solutions:**

1. **Enable Qdrant in service configuration:**
```bash
# Edit .vrooli/service.json
jq '.dependencies.resources.qdrant.enabled = true' .vrooli/service.json > tmp.json && mv tmp.json .vrooli/service.json

# Re-run setup
vrooli develop --setup
```

2. **Manual Qdrant installation:**
```bash
# Install Qdrant resource
vrooli setup --resources qdrant --yes yes

# Verify installation
resource-qdrant help
```

### Issue: Qdrant Server Not Running

**Symptoms:**
```bash
$ curl http://localhost:6333/
curl: (7) Failed to connect to localhost port 6333: Connection refused
```

**Diagnosis:**
```bash
# Check if Qdrant process is running
ps aux | grep qdrant

# Check Qdrant logs
docker logs qdrant 2>&1 | tail -20

# Check port availability
netstat -tlnp | grep 6333
```

**Solutions:**

1. **Start Qdrant service:**
```bash
# Using Docker
docker start qdrant

# Or restart if needed
docker restart qdrant

# Verify startup
curl http://localhost:6333/
```

2. **Check Docker configuration:**
```bash
# Verify Qdrant container exists
docker ps -a | grep qdrant

# If container doesn't exist, recreate
docker run -d --name qdrant -p 6333:6333 qdrant/qdrant:latest
```

3. **Check firewall and network:**
```bash
# Test local connectivity
telnet localhost 6333

# Check if port is bound
sudo lsof -i :6333
```

### Issue: Permission Denied Errors

**Symptoms:**
```bash
$ resource-qdrant embeddings init
Permission denied: cannot create .vrooli/app-identity.json
```

**Diagnosis:**
```bash
# Check directory permissions
ls -la .vrooli/
ls -la resources/qdrant/

# Check file ownership
stat .vrooli/
```

**Solutions:**

1. **Fix directory permissions:**
```bash
# Create .vrooli directory if missing
mkdir -p .vrooli

# Fix ownership (replace with your username)
sudo chown -R $USER:$USER .vrooli/
chmod 755 .vrooli/

# Fix script permissions
chmod +x resources/qdrant/cli.sh
chmod +x resources/qdrant/embeddings/cli.sh
```

2. **SELinux issues (if applicable):**
```bash
# Check SELinux status
sestatus

# Set appropriate contexts
setsebool -P httpd_can_network_connect on
```

## 🔍 Search and Query Issues

### Issue: No Search Results Found

**Symptoms:**
```bash
$ resource-qdrant search "email notification"
No results found
```

**Diagnosis:**
```bash
# Check if embeddings exist
resource-qdrant embeddings status

# Check collections in Qdrant
curl -s http://localhost:6333/collections | jq '.result[] | {name: .name, vectors: .vectors_count}'

# Validate app identity
cat .vrooli/app-identity.json | jq
```

**Solutions:**

1. **Initialize and refresh embeddings:**
```bash
# Initialize if not done
resource-qdrant embeddings init

# Refresh embeddings
resource-qdrant embeddings refresh

# Verify collections created
curl -s http://localhost:6333/collections | jq '.result[].name'
```

2. **Check content extraction:**
```bash
# Validate what content is being found
resource-qdrant embeddings validate

# Test specific extractors
# Use the v2.0 compliant CLI
resource-qdrant-embeddings <command>
qdrant::extract::find_workflows "."
qdrant::extract::find_docs "."
```

3. **Lower search threshold:**
```bash
# Try with lower similarity threshold
resource-qdrant search "email notification" workflows 10 0.5
```

### Issue: Search Results Not Relevant

**Symptoms:**
- Search returns results but they're not relevant to the query
- Low similarity scores across all results

**Diagnosis:**
```bash
# Check search parameters
resource-qdrant search "test query" workflows 5 0.8 --verbose

# Test with different content types
resource-qdrant search "test query" code
resource-qdrant search "test query" knowledge
```

**Solutions:**

1. **Adjust search parameters:**
```bash
# Try broader search
resource-qdrant search "email" # Search all types
resource-qdrant search "email notification" workflows 20 0.6

# Use different query phrasing
resource-qdrant search "send automated emails"
resource-qdrant search "email delivery system"
```

2. **Check embedding model:**
```bash
# Verify embedding model in use
grep "DEFAULT_MODEL" resources/qdrant/embeddings/config/unified.sh

# Consider switching models (set environment variable)
# DEFAULT_MODEL="nomic-embed-text"  # More semantic
# DEFAULT_MODEL="mxbai-embed-large"  # Highest quality
```

3. **Refresh embeddings with different model:**
```bash
# Set model via environment variable
export QDRANT_EMBEDDING_MODEL_OVERRIDE="nomic-embed-text"
# Or edit config/unified.sh

# Force refresh with new model
resource-qdrant embeddings refresh --force
```

### Issue: Search Too Slow

**Symptoms:**
- Searches take more than 2-3 seconds
- Multi-app searches timeout

**Diagnosis:**
```bash
# Time search operations
time resource-qdrant search "test query"
time resource-qdrant search-all "test query"

# Check system resources
top -p $(pgrep qdrant)
free -h
```

**Solutions:**

1. **Optimize search parameters:**
```bash
# Use smaller result limits
resource-qdrant search "query" workflows 5

# Search specific types instead of all
resource-qdrant search "query" code
```

2. **Check Qdrant configuration:**
```bash
# View current Qdrant configuration
curl -s http://localhost:6333/collections/test-collection | jq '.result.config'

# Optimize HNSW parameters (see PERFORMANCE.md)
```

3. **Implement caching:**
```bash
# Enable Redis caching if available
redis-cli ping

# Check if caching is working
redis-cli keys "search:*"
```

## 📁 Content Extraction Issues

### Issue: Content Not Being Extracted

**Symptoms:**
```bash
$ resource-qdrant embeddings validate
No workflows found
No scenarios found
```

**Diagnosis:**
```bash
# Check file structure
find . -name "*.json" -path "*/initialization/*" | head -5
find . -name "PRD.md" -path "*/scenarios/*" | head -5
find . -name "*.md" -path "*/docs/*" | head -5

# Test extractors manually
# Use the v2.0 compliant CLI
resource-qdrant-embeddings <command>
qdrant::extract::find_workflows "."
```

**Solutions:**

1. **Check file patterns and locations:**
```bash
# Workflows should be in: */initialization/automation/n8n/*.json
ls -la initialization/automation/n8n/

# Scenarios should have: PRD.md and .scenario.yaml
ls -la scenarios/*/

# Docs should be in: docs/*.md
ls -la docs/
```

2. **Fix file structure:**
```bash
# Create missing directories
mkdir -p initialization/automation/n8n
mkdir -p scenarios/sample-app
mkdir -p docs

# Move files to correct locations if needed
mv workflow.json initialization/automation/n8n/
mv README.md docs/ARCHITECTURE.md
```

3. **Test individual extractors:**
```bash
# Test each extractor individually
# Use the v2.0 compliant CLI
resource-qdrant-embeddings <command>

# Test workflows extractor
qdrant::extract::workflows_batch "." "/tmp/test_workflows.txt"
cat /tmp/test_workflows.txt

# Test docs extractor
qdrant::extract::docs_batch "." "/tmp/test_docs.txt"
cat /tmp/test_docs.txt
```

### Issue: Extractor Errors

**Symptoms:**
```bash
$ resource-qdrant embeddings refresh
Error: Failed to extract workflows
Error: jq: parse error: Invalid JSON
```

**Diagnosis:**
```bash
# Check for malformed JSON files
find . -name "*.json" -exec sh -c 'echo "Checking $1"; jq empty "$1" || echo "Invalid JSON: $1"' _ {} \;

# Check for missing dependencies
which jq yq curl
```

**Solutions:**

1. **Fix JSON syntax errors:**
```bash
# Find and fix invalid JSON
find . -name "*.json" -path "*/initialization/*" -exec sh -c '
    if ! jq empty "$1" 2>/dev/null; then
        echo "Fixing JSON: $1"
        # Backup original
        cp "$1" "$1.backup"
        # Try to fix common issues
        sed -i "s/,\s*}/}/g" "$1"  # Remove trailing commas
        sed -i "s/,\s*]/]/g" "$1"  # Remove trailing commas in arrays
    fi
' _ {} \;
```

2. **Install missing dependencies:**
```bash
# Install jq if missing
sudo apt-get install jq

# Install yq if missing
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
sudo chmod +x /usr/local/bin/yq
```

3. **Check extractor script syntax:**
```bash
# Validate extractor scripts
bash -n resources/qdrant/embeddings/extractors/workflows.sh
bash -n resources/qdrant/embeddings/extractors/scenarios.sh
```

## 🗄️ Database and Storage Issues

### Issue: Qdrant Connection Errors

**Symptoms:**
```bash
$ resource-qdrant search "test"
Error: Failed to connect to Qdrant
curl: (7) Failed to connect to localhost port 6333
```

**Diagnosis:**
```bash
# Check Qdrant service status
docker ps | grep qdrant
systemctl status qdrant  # If running as service

# Check network connectivity
ping localhost
telnet localhost 6333

# Check Qdrant logs
docker logs qdrant 2>&1 | tail -20
```

**Solutions:**

1. **Restart Qdrant service:**
```bash
# Docker restart
docker restart qdrant

# Wait for startup
sleep 10

# Verify health
curl http://localhost:6333/
```

2. **Check Docker networking:**
```bash
# Check Docker network
docker network ls
docker inspect qdrant | jq '.[0].NetworkSettings'

# Recreate container with correct ports
docker stop qdrant
docker rm qdrant
docker run -d --name qdrant -p 6333:6333 -v $(pwd)/qdrant_storage:/qdrant/storage qdrant/qdrant:latest
```

3. **Firewall and port issues:**
```bash
# Check if port is in use
sudo netstat -tlnp | grep 6333

# Check firewall rules
sudo ufw status
sudo iptables -L | grep 6333

# Open port if needed
sudo ufw allow 6333
```

### Issue: Collection Creation Failures

**Symptoms:**
```bash
$ resource-qdrant embeddings init
Error: Failed to create collection app-workflows
HTTP 400: Collection already exists
```

**Diagnosis:**
```bash
# Check existing collections
curl -s http://localhost:6333/collections | jq '.result[].name'

# Check collection configuration
curl -s http://localhost:6333/collections/app-workflows | jq '.result.config'
```

**Solutions:**

1. **Clean up existing collections:**
```bash
# List all collections for app
app_id=$(cat .vrooli/app-identity.json | jq -r '.app_id')
curl -s http://localhost:6333/collections | jq -r ".result[] | select(.name | startswith(\"$app_id\")) | .name"

# Delete specific collection
curl -X DELETE "http://localhost:6333/collections/app-workflows"

# Or use garbage collection
resource-qdrant embeddings gc --force
```

2. **Force recreation:**
```bash
# Remove app identity and reinitialize
rm .vrooli/app-identity.json
resource-qdrant embeddings init
```

3. **Check vector dimensions mismatch:**
```bash
# Check current collection dimensions
curl -s http://localhost:6333/collections/app-workflows | jq '.result.config.params.vectors.size'

# If dimensions changed, recreate collection
curl -X DELETE "http://localhost:6333/collections/app-workflows"
resource-qdrant embeddings init
```

### Issue: Disk Space Problems

**Symptoms:**
```bash
$ resource-qdrant embeddings refresh
Error: No space left on device
df: /qdrant/storage: No space left on device
```

**Diagnosis:**
```bash
# Check disk usage
df -h
du -sh qdrant_storage/

# Check Qdrant storage location
docker inspect qdrant | jq '.[0].Mounts'
```

**Solutions:**

1. **Clean up old data:**
```bash
# Remove old collections
resource-qdrant embeddings gc --force

# Vacuum Qdrant database
curl -X POST "http://localhost:6333/collections/app-workflows/index" \
    -H "Content-Type: application/json" \
    -d '{"wait": true}'
```

2. **Move storage location:**
```bash
# Stop Qdrant
docker stop qdrant

# Create new storage location
sudo mkdir -p /opt/qdrant/storage
sudo chown $USER:$USER /opt/qdrant/storage

# Restart with new location
docker rm qdrant
docker run -d --name qdrant -p 6333:6333 -v /opt/qdrant/storage:/qdrant/storage qdrant/qdrant:latest
```

3. **Implement data retention:**
```bash
# Create cleanup script
cat > cleanup-embeddings.sh << 'EOF'
#!/usr/bin/env bash
# Remove embeddings older than 30 days
find qdrant_storage/ -type f -mtime +30 -delete

# Compact collections
curl -X POST "http://localhost:6333/collections/app-workflows/index" \
    -H "Content-Type: application/json" \
    -d '{"wait": true}'
EOF

chmod +x cleanup-embeddings.sh

# Add to crontab
(crontab -l 2>/dev/null; echo "0 2 * * 0 $(pwd)/cleanup-embeddings.sh") | crontab -
```

## 🚀 Performance Issues

### Issue: Slow Embedding Generation

**Symptoms:**
- Embedding refresh takes more than 5 minutes for small projects
- CPU usage consistently high during refresh

**Diagnosis:**
```bash
# Monitor resource usage during refresh
top -p $(pgrep ollama)
iostat -x 1

# Check embedding model
grep "DEFAULT_MODEL" resources/qdrant/embeddings/config/unified.sh

# Time individual operations
time resource-qdrant embeddings refresh
```

**Solutions:**

1. **Use faster embedding model:**
```bash
# Set faster model via environment variable
export QDRANT_EMBEDDING_MODEL_OVERRIDE="all-minilm"
# Or edit config/unified.sh

# Refresh with new model
resource-qdrant embeddings refresh --force
```

2. **Optimize batch processing:**
```bash
# Reduce batch size for memory-constrained systems
export EMBEDDING_BATCH_SIZE=5

# Or increase for powerful systems
export EMBEDDING_BATCH_SIZE=20
```

3. **Check Ollama configuration:**
```bash
# Verify Ollama is running efficiently
ollama list
ollama show mxbai-embed-large

# Restart Ollama if needed
docker restart ollama
```

### Issue: Memory Usage Too High

**Symptoms:**
```bash
$ free -h
              total        used        free
Mem:           8.0G        7.8G        200M
Swap:          2.0G        1.5G        500M
```

**Solutions:**

1. **Reduce memory usage:**
```bash
# Use smaller embedding model
export DEFAULT_MODEL="all-minilm"  # 384 dimensions vs 1024

# Reduce batch sizes
export EMBEDDING_BATCH_SIZE=5

# Enable memory mapping in Qdrant
curl -X PATCH "http://localhost:6333/collections/app-workflows" \
    -H "Content-Type: application/json" \
    -d '{"optimizers_config": {"memmap_threshold": 10000}}'
```

2. **Clean up resources:**
```bash
# Clear embedding cache
redis-cli FLUSHDB

# Garbage collect old collections
resource-qdrant embeddings gc --force

# Restart services
docker restart qdrant ollama
```

## 🛠️ Advanced Troubleshooting

### Issue: Corrupted App Identity

**Symptoms:**
```bash
$ cat .vrooli/app-identity.json
{"app_id": null, "last_indexed": null, ...}
```

**Solutions:**
```bash
# Backup current identity
cp .vrooli/app-identity.json .vrooli/app-identity.json.backup

# Regenerate identity
rm .vrooli/app-identity.json
resource-qdrant embeddings init

# If you need to preserve specific settings
jq --arg app_id "my-app-v2" '.app_id = $app_id' .vrooli/app-identity.json > tmp.json && mv tmp.json .vrooli/app-identity.json
```

### Issue: Inconsistent Search Results

**Symptoms:**
- Same query returns different results each time
- Results appear and disappear randomly

**Diagnosis:**
```bash
# Check collection consistency
curl -s "http://localhost:6333/collections/app-workflows/points/scroll" \
    -H "Content-Type: application/json" \
    -d '{"limit": 10}' | jq '.result.points | length'

# Verify vector integrity
curl -s "http://localhost:6333/collections/app-workflows" | jq '.result.status'
```

**Solutions:**
```bash
# Rebuild collections from scratch
resource-qdrant embeddings gc --force
rm .vrooli/app-identity.json
resource-qdrant embeddings init
resource-qdrant embeddings refresh --force

# Optimize collections
curl -X POST "http://localhost:6333/collections/app-workflows/index" \
    -H "Content-Type: application/json" \
    -d '{"wait": true}'
```

### Issue: CLI Commands Not Working

**Symptoms:**
```bash
$ resource-qdrant
Error: Unknown command 'resource-qdrant'
```

**Solutions:**
```bash
# Check Vrooli CLI installation
vrooli help

# Verify resource-qdrant script exists
ls -la resources/qdrant/cli.sh

# Check PATH and permissions
echo $PATH
chmod +x resources/qdrant/cli.sh

# Source Vrooli environment
source ~/.bashrc
# or
source ~/.zshrc
```

## 📋 Troubleshooting Checklist

### When Search Returns No Results

- [ ] Check if Qdrant is running: `curl http://localhost:6333/`
- [ ] Verify collections exist: `curl -s http://localhost:6333/collections | jq '.result[].name'`
- [ ] Check app identity: `cat .vrooli/app-identity.json`
- [ ] Validate content extraction: `resource-qdrant embeddings validate`
- [ ] Try broader search terms
- [ ] Lower similarity threshold
- [ ] Force refresh embeddings

### When Embedding Refresh Fails

- [ ] Check disk space: `df -h`
- [ ] Verify Qdrant connectivity: `curl http://localhost:6333/`
- [ ] Check file permissions: `ls -la .vrooli/`
- [ ] Validate JSON syntax in workflow files
- [ ] Check for malformed documents
- [ ] Review extractor script errors
- [ ] Try incremental refresh

### When Performance Is Poor

- [ ] Check system resources: `top`, `free -h`
- [ ] Monitor Qdrant performance
- [ ] Consider smaller embedding model
- [ ] Reduce batch sizes
- [ ] Enable result caching
- [ ] Optimize Qdrant configuration
- [ ] Check network latency to Qdrant

### When Git Integration Fails

- [ ] Verify `cli.sh` has refresh function
- [ ] Check app identity has `index_commit`
- [ ] Test manual refresh works
- [ ] Verify develop command integration
- [ ] Check debug logs
- [ ] Force refresh as workaround

## 🆘 Emergency Recovery

### Complete System Reset

```bash
#!/usr/bin/env bash
# Emergency reset script - use when everything is broken

echo "=== EMERGENCY SEMANTIC KNOWLEDGE SYSTEM RESET ==="
echo "This will delete all embeddings and start fresh."
read -p "Are you sure? (type 'yes' to continue): " confirm

if [[ "$confirm" != "yes" ]]; then
    echo "Aborted."
    exit 1
fi

# 1. Stop all services
echo "Stopping services..."
docker stop qdrant ollama 2>/dev/null || true

# 2. Backup current state
echo "Backing up current state..."
mkdir -p backup/$(date +%Y%m%d_%H%M%S)
cp -r .vrooli/ backup/$(date +%Y%m%d_%H%M%S)/ 2>/dev/null || true

# 3. Clean slate
echo "Removing all embeddings data..."
rm -rf .vrooli/app-identity.json
redis-cli FLUSHDB 2>/dev/null || true

# 4. Delete all collections
echo "Removing Qdrant collections..."
for collection in $(curl -s http://localhost:6333/collections | jq -r '.result[]?.name // empty' 2>/dev/null); do
    curl -X DELETE "http://localhost:6333/collections/$collection" 2>/dev/null || true
done

# 5. Restart services
echo "Restarting services..."
docker restart qdrant ollama 2>/dev/null || true
sleep 10

# 6. Reinitialize
echo "Reinitializing embedding system..."
resource-qdrant embeddings init

# 7. Fresh refresh
echo "Performing fresh embedding refresh..."
resource-qdrant embeddings refresh --force

echo "=== RESET COMPLETE ==="
echo "Check status with: resource-qdrant embeddings status"
```

### Gradual Recovery

```bash
# Step 1: Basic connectivity
curl http://localhost:6333/ || echo "Qdrant not responding"

# Step 2: Service restart
docker restart qdrant
sleep 10

# Step 3: Clean collections
resource-qdrant embeddings gc --force

# Step 4: Reinitialize app
rm .vrooli/app-identity.json
resource-qdrant embeddings init

# Step 5: Selective refresh
resource-qdrant embeddings refresh --force

# Step 6: Validate results
resource-qdrant embeddings validate
resource-qdrant search "test query"
```

## 📞 Getting Help

### Diagnostic Information to Collect

When reporting issues, include:

```bash
# System information
echo "=== SYSTEM INFO ==="
uname -a
free -h
df -h

# Qdrant status
echo "=== QDRANT STATUS ==="
curl -s http://localhost:6333/ 2>&1
curl -s http://localhost:6333/collections 2>&1 | jq '.result[].name' 2>&1

# App identity
echo "=== APP IDENTITY ==="
cat .vrooli/app-identity.json 2>&1

# Service configuration
echo "=== SERVICE CONFIG ==="
cat .vrooli/service.json | jq '.dependencies.resources.qdrant' 2>&1

# Recent logs
echo "=== RECENT LOGS ==="
docker logs qdrant 2>&1 | tail -20
```

### Common Error Patterns

| Error Pattern | Likely Cause | Quick Fix |
|---------------|-------------|-----------|
| `Connection refused` | Qdrant not running | `docker restart qdrant` |
| `Permission denied` | File permissions | `chmod 755 .vrooli/` |
| `No space left` | Disk full | `resource-qdrant embeddings gc` |
| `Invalid JSON` | Malformed workflow files | Validate and fix JSON syntax |
| `Collection exists` | Stale collections | `rm .vrooli/app-identity.json && init` |
| `No results found` | No embeddings | `resource-qdrant embeddings refresh` |

---

*For additional support, check the project documentation or create an issue with the diagnostic information above.*
