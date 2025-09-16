# Eclipse Ditto - Known Problems and Solutions

## Critical Issues

### 1. Gateway Clustering Requirement (BLOCKER)
**Problem**: Eclipse Ditto 3.5.6 gateway service fails to start in standalone mode due to mandatory cluster formation requirements.

**Symptoms**:
- Gateway container continuously restarts
- Logs show: "Discovered [0] contact points, confirmed [0], which is less than the required [2]"
- API endpoints not accessible on configured port

**Attempted Solutions**:
1. Set `REMOTE_SEEDS=` environment variable - **Failed**
2. Added `-Dpekko.cluster.seed-nodes=` to JAVA_TOOL_OPTIONS - **Failed**
3. Configured service endpoints manually - **Failed**
4. Disabled cluster DNS discovery - **Failed**

**Root Cause**: 
The Pekko (formerly Akka) clustering system requires minimum 2 seed nodes for quorum, even when trying to run standalone. The gateway service creates a ClusterActorRefProvider which enforces this requirement.

**Potential Solutions**:
- Use an older version of Ditto that supports true standalone mode
- Deploy multiple instances to satisfy cluster requirements
- Modify Ditto configuration files to bypass cluster checks (requires custom image)
- Use Ditto's "devops" deployment mode instead of production mode

### 2. Port Conflicts
**Problem**: Default port 8089 often conflicts with other services.

**Solution**: Changed to port 8094 in configuration files.

### 3. Health Check Endpoints
**Problem**: Documentation shows various health check endpoints that don't work.

**Attempted Endpoints**:
- `/health` - Returns 404
- `/alive` - Returns 404
- `/status/health` - Connection refused

**Current Status**: No working health check endpoint identified for standalone deployment.

## Configuration Challenges

### Docker Compose Structure
The official docker-compose.yml from Eclipse Ditto repository is designed for clustered deployments. Attempts to simplify for standalone use have been unsuccessful due to hardcoded clustering requirements in the application.

### Environment Variables
Critical environment variables that affect clustering:
- `REMOTE_SEEDS` - Setting to empty doesn't disable clustering
- `DITTO_CLUSTER_ENABLED` - Not recognized
- `BIND_HOSTNAME` - Setting to 0.0.0.0 doesn't help
- `ENABLE_PRE_AUTHENTICATION` - Doesn't affect clustering

## Recommendations for Future Improvements

1. **Investigation Needed**: Research if Eclipse Ditto has a true standalone mode in newer versions or if there's a specific configuration flag to disable clustering entirely.

2. **Alternative Deployment**: Consider using Kubernetes with minimum replicas or Docker Swarm to satisfy cluster requirements.

3. **Custom Image**: Build a custom Ditto image with modified configuration files that bypass cluster checks.

4. **Version Downgrade**: Try Eclipse Ditto 2.x series which may have better standalone support.

## Test Results Summary

- **Docker Deployment**: ✅ Works (containers start)
- **Service Health**: ❌ Gateway unhealthy due to clustering
- **API Access**: ❌ Not accessible
- **CLI Integration**: ✅ Commands structured correctly
- **WebSocket**: ❌ Requires working gateway
- **Authentication**: ❌ Requires working gateway

## Next Steps for Resolution

1. Research Eclipse Ditto documentation for standalone deployment options
2. Try older version (2.4.x) that may support standalone better
3. Consider minimal cluster setup with 2+ gateway instances
4. Investigate "development mode" configurations
5. Check if upstream project has addressed this in recent releases