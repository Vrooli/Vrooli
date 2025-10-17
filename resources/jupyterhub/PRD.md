# JupyterHub Resource PRD

## Executive Summary
**What**: Multi-user server that spawns, manages, and proxies multiple instances of Jupyter notebook servers for collaborative computing
**Why**: Enables teams to collaborate on data science, machine learning experiments, and interactive documentation with live code
**Who**: Data science teams, ML engineers, researchers, educators, analysts
**Value**: Enables $100K+ in collaborative data science and educational capabilities
**Priority**: P0 - Core interactive computing infrastructure

## Requirements Checklist

### P0 Requirements (Must Have)
- [ ] **v2.0 Universal Contract Compliance**: Full implementation of all required CLI commands
  - Test: `resource-jupyterhub help | grep -E "manage|test|content"`
  - Validation: All command groups present and functional
  
- [ ] **Health Check System**: Responds within 5 seconds with meaningful status
  - Test: `timeout 5 curl -sf http://localhost:8000/hub/health`
  - Validation: Returns JSON with hub and proxy status
  
- [ ] **Multi-User Authentication**: OAuth-based authentication system
  - Test: `curl -sf http://localhost:8000/hub/api/users | jq .`
  - Validation: User list accessible via API
  
- [ ] **Spawner Management**: Spawn and manage individual notebook servers
  - Test: `resource-jupyterhub content spawn --user testuser`
  - Validation: Creates isolated notebook environment
  
- [ ] **Proxy Configuration**: Route traffic to individual notebook instances
  - Test: `curl -sf http://localhost:8000/hub/api/proxy | jq .`
  - Validation: Proxy routes configured correctly

### P1 Requirements (Should Have)
- [ ] **Resource Limits**: CPU/memory limits per user
  - Test: `resource-jupyterhub content limits --user testuser`
  - Validation: Shows configured resource limits
  
- [ ] **Persistent Storage**: User workspaces persist across sessions
  - Test: `resource-jupyterhub content list --user testuser`
  - Validation: Shows user's saved notebooks
  
- [ ] **Extension Management**: Install and manage JupyterLab extensions
  - Test: `resource-jupyterhub content extensions list`
  - Validation: Lists available extensions

### P2 Requirements (Nice to Have)
- [ ] **GPU Support**: Enable GPU access for ML workloads
  - Test: `resource-jupyterhub info --json | jq .features.gpu`
  - Validation: Shows GPU availability
  
- [ ] **Custom Environments**: User-defined conda/pip environments
  - Test: `resource-jupyterhub content environments list`
  - Validation: Shows available environments

## Technical Specifications

### Architecture
- **Core Components**:
  - JupyterHub: Central authentication and spawning service
  - Configurable Proxy: Routes requests to user servers
  - Spawner: Creates isolated notebook instances (DockerSpawner)
  - Authenticator: OAuth/GitHub/Google authentication
  
- **Ports**:
  - 8000: JupyterHub main interface
  - 8001: Hub API endpoint
  - 8081: Proxy API endpoint
  
- **Dependencies**:
  - postgres: User database and session storage
  - redis: Session caching and real-time updates
  - docker: Container-based user isolation
  
- **Storage**:
  - User notebooks: `/data/resources/jupyterhub/users/{username}`
  - Shared data: `/data/resources/jupyterhub/shared`
  - Extensions: `/data/resources/jupyterhub/extensions`

### Performance Requirements
- **Startup Time**: <45 seconds for hub initialization
- **Spawn Time**: <30 seconds per user server
- **Concurrent Users**: Support 50+ simultaneous users
- **API Response**: <500ms for management operations

### Security Configuration
- **Authentication Methods**:
  - OAuth 2.0 (GitHub, Google, GitLab)
  - Native JupyterHub authentication
  - LDAP/Active Directory (optional)
  
- **Authorization**:
  - Role-based access (admin, user, instructor)
  - Group-based permissions
  - Resource quotas per user/group
  
- **Network Security**:
  - HTTPS with auto-generated certificates
  - User isolation via Docker containers
  - Network policies between user servers

## Integration Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: User database and hub state storage
    integration_pattern: Direct SQL connection
    database: jupyterhub_db
    
  - resource_name: docker
    purpose: Container-based user isolation
    integration_pattern: Docker socket mounting
    spawner: DockerSpawner
    
optional:
  - resource_name: redis
    purpose: Session caching and real-time updates
    integration_pattern: Redis connection for performance
    fallback: In-memory caching
    
  - resource_name: ollama
    purpose: AI model access within notebooks
    integration_pattern: API endpoint exposure
    enable_for: ML/AI workflows
```

### Integration Patterns
```yaml
scenarios_using_resource:
  - scenario_name: ml-training-platform
    usage_pattern: Collaborative ML experiment tracking
    features: GPU access, shared datasets, model registry
    
  - scenario_name: data-analytics-workspace
    usage_pattern: Team-based data analysis and visualization
    features: SQL access, shared libraries, dashboards
    
  - scenario_name: educational-platform
    usage_pattern: Interactive programming courses
    features: Assignment distribution, auto-grading, progress tracking
    
resource_integrations:
  - postgres → jupyterhub: User authentication and state
  - jupyterhub → docker: Spawn isolated containers
  - jupyterhub → ollama: AI model inference in notebooks
  - jupyterhub → minio: Object storage for large datasets
```

### API Endpoints
```yaml
hub_api:
  - method: GET
    path: /hub/api/users
    purpose: List all users and their server status
    
  - method: POST
    path: /hub/api/users/{username}/server
    purpose: Start a user's notebook server
    
  - method: DELETE
    path: /hub/api/users/{username}/server
    purpose: Stop a user's notebook server
    
  - method: GET
    path: /hub/api/proxy
    purpose: Get proxy routing table
    
admin_api:
  - method: POST
    path: /hub/api/users
    purpose: Create new user
    
  - method: PUT
    path: /hub/api/users/{username}
    purpose: Update user permissions
    
  - method: GET
    path: /hub/api/groups
    purpose: List user groups
```

## Configuration Schema

### JupyterHub Configuration
```python
# jupyterhub_config.py structure
c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'
c.JupyterHub.hub_ip = '0.0.0.0'
c.JupyterHub.hub_port = 8001

# Authentication
c.JupyterHub.authenticator_class = 'oauthenticator.generic.GenericOAuthenticator'
c.OAuthenticator.client_id = os.environ.get('OAUTH_CLIENT_ID')
c.OAuthenticator.client_secret = os.environ.get('OAUTH_CLIENT_SECRET')

# Spawner configuration
c.DockerSpawner.image = 'jupyter/scipy-notebook:latest'
c.DockerSpawner.network_name = 'vrooli-network'
c.DockerSpawner.cpu_limit = 2
c.DockerSpawner.mem_limit = '4G'

# Persistent storage
c.DockerSpawner.volumes = {
    '/data/resources/jupyterhub/users/{username}': '/home/jovyan/work'
}
```

### Environment Variables
```yaml
environment_variables:
  - var: JUPYTERHUB_CRYPT_KEY
    purpose: Encrypt authentication cookies
    generation: Automatic on first install
    
  - var: OAUTH_CLIENT_ID
    purpose: OAuth application client ID
    required_for: OAuth authentication
    
  - var: OAUTH_CLIENT_SECRET
    purpose: OAuth application client secret
    required_for: OAuth authentication
    
  - var: POSTGRES_CONNECTION_STRING
    purpose: Database connection for hub state
    format: postgresql://user:pass@host/jupyterhub_db
    
  - var: DOCKER_NETWORK_NAME
    purpose: Docker network for user containers
    default: vrooli-network
```

## Operational Requirements

### Deployment Architecture
```yaml
containerization:
  hub_image: jupyterhub/jupyterhub:latest
  notebook_images:
    - jupyter/base-notebook: Minimal Python environment
    - jupyter/scipy-notebook: Scientific Python stack
    - jupyter/tensorflow-notebook: Deep learning stack
    - jupyter/datascience-notebook: R, Julia, Python
    
networking:
  required_networks:
    - vrooli-network: Primary inter-resource communication
    - jupyterhub-network: Internal hub-to-user communication
    
  port_allocation:
    hub_port: 8000  # Main web interface
    api_port: 8001  # Hub API
    proxy_port: 8081  # Proxy API
    user_range: 9000-9999  # Dynamic user server ports
    
data_management:
  persistence:
    - volume: jupyterhub-data
      mount: /data/jupyterhub
      purpose: Hub configuration and state
      
    - volume: user-notebooks
      mount: /data/resources/jupyterhub/users
      purpose: User notebooks and data
      
  backup_strategy:
    method: Volume snapshots
    frequency: Daily
    retention: 30 days
```

### Performance Optimization
```yaml
scaling_configuration:
  horizontal_scaling:
    max_users_per_hub: 100
    hub_replication: Kubernetes-based scaling
    
  vertical_scaling:
    hub_resources:
      min_cpu: 1
      max_cpu: 4
      min_memory: 2G
      max_memory: 8G
      
  caching:
    redis_cache: User sessions and auth tokens
    proxy_cache: Route caching for performance
    
resource_limits:
  per_user_defaults:
    cpu_limit: 2.0
    memory_limit: 4G
    storage_limit: 10G
    
  configurable_profiles:
    - name: small
      cpu: 1.0
      memory: 2G
      
    - name: medium
      cpu: 2.0
      memory: 4G
      
    - name: large
      cpu: 4.0
      memory: 8G
```

## Testing Strategy

### Test Implementation
```yaml
unit_tests:
  location: lib/*.bats (co-located with shell scripts)
  coverage: Configuration validation, API wrappers
  
integration_tests:
  location: test/phases/
  coverage:
    - User authentication flow
    - Server spawning and stopping
    - Resource limit enforcement
    - Proxy routing validation
    
smoke_tests:
  - Hub health check responds
  - Proxy is accessible
  - Can spawn test user server
  - API authentication works
  
performance_tests:
  - Spawn time under load (50 users)
  - API response times
  - Resource consumption per user
  - Hub recovery after crash
```

## Infrastructure Value

### Enabling Capabilities
1. **Collaborative Data Science**: Teams work on shared projects with isolated environments
2. **ML Experimentation**: GPU-enabled notebooks for model training
3. **Educational Platforms**: Interactive programming courses with auto-grading
4. **Research Computing**: Reproducible computational research environments
5. **Analytics Workspaces**: Business intelligence and data visualization

### Scenario Amplification
- **Data Science Scenarios**: +$50K value from collaborative features
- **ML Training Scenarios**: +$30K value from GPU access and scaling
- **Educational Scenarios**: +$40K value from multi-user management
- **Research Scenarios**: +$25K value from reproducible environments

### Resource Economics
- **Setup Cost**: 2 hours initial configuration
- **Operating Cost**: 4GB RAM + 2 CPU cores base (scales with users)
- **Integration Value**: 10x productivity boost for data teams
- **Maintenance**: Monthly security updates, extension management

## Implementation Notes

### Critical Success Factors
1. **Authentication Setup**: OAuth configuration must be completed
2. **Storage Strategy**: User data persistence is essential
3. **Resource Limits**: Prevent runaway resource consumption
4. **Network Isolation**: Security between user environments
5. **Backup Strategy**: User notebooks must be protected

### Known Challenges
- **Spawner Timeouts**: Large images take time to pull initially
  - Solution: Pre-pull common images during installation
  
- **Resource Contention**: Many users can overwhelm single hub
  - Solution: Implement resource quotas and profiles
  
- **Extension Conflicts**: Some extensions incompatible
  - Solution: Curated extension list with compatibility matrix

### Integration Considerations
- **PostgreSQL Required**: Hub won't start without database
- **Docker Socket Access**: Spawner needs Docker permissions
- **Network Configuration**: Proxy must reach user containers
- **SSL Certificates**: HTTPS recommended for production

## Validation Criteria

### Infrastructure Validation
- [ ] Hub starts and responds to health checks
- [ ] OAuth authentication configured and working
- [ ] User servers spawn successfully
- [ ] Proxy routes traffic correctly
- [ ] Resource limits enforced

### Integration Validation
- [ ] PostgreSQL connection established
- [ ] Docker spawner creates containers
- [ ] User data persists across sessions
- [ ] Extensions install correctly
- [ ] API endpoints accessible

### Security Validation
- [ ] User isolation verified
- [ ] Authentication required for all endpoints
- [ ] Resource quotas prevent abuse
- [ ] Network policies enforced
- [ ] Sensitive data encrypted

## Progress Tracking

### Implementation Status
- **P0 Completion**: 0% → Target 100%
- **P1 Completion**: 0% → Target 75%
- **P2 Completion**: 0% → Target 25%
- **Overall Progress**: 0% → Target 85%

### Milestones
1. **Phase 1**: Basic hub with authentication (Week 1)
2. **Phase 2**: Docker spawner integration (Week 2)
3. **Phase 3**: Persistent storage and limits (Week 3)
4. **Phase 4**: Extensions and optimizations (Week 4)

---

**Last Updated**: 2025-01-10
**Status**: Draft
**Owner**: Ecosystem Manager
**Review Cycle**: Weekly during implementation