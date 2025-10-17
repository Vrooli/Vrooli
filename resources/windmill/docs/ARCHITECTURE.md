# Windmill Architecture

This document describes the architecture and internal design of Windmill.

## System Overview

Windmill follows a distributed architecture with these core components:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Web UI    │────▶│  API Server │────▶│  PostgreSQL │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                    ┌──────▼──────┐
                    │    Redis    │
                    └──────┬──────┘
                           │
        ┌──────────────────┴──────────────────┐
        │                                      │
  ┌─────▼─────┐  ┌─────────────┐  ┌──────────▼──┐
  │  Worker 1  │  │  Worker 2   │  │  Worker N   │
  └───────────┘  └─────────────┘  └─────────────┘
```

## Core Services

### 1. Windmill Server

The main API server that handles:
- REST API endpoints
- WebSocket connections for real-time updates
- Authentication and authorization
- Job scheduling and orchestration
- Resource management

**Port**: 5681 (default)  
**Container**: `windmill-server`

### 2. Workers

Distributed execution units that:
- Execute scripts, flows, and apps
- Support multiple languages (Python, TypeScript, Go, Bash)
- Handle resource isolation
- Report status back to server

**Scaling**: 1-N workers  
**Container**: `windmill-worker-*`

### 3. PostgreSQL Database

Primary data store for:
- Job definitions and history
- User accounts and permissions
- Scripts, flows, and apps
- Execution logs and results

**Port**: 5432 (internal)  
**Container**: `windmill-postgres`

### 4. Redis

Cache and message broker for:
- Job queue management
- Real-time event streaming
- Session storage
- Temporary data caching

**Port**: 6379 (internal)  
**Container**: `windmill-redis`

### 5. Language Server Protocol (LSP)

Optional service providing:
- Code completion
- Syntax checking
- IntelliSense features

**Memory Usage**: ~500MB per language

## Storage Architecture

### Directory Structure

```
/opt/windmill/
├── data/
│   ├── scripts/         # Script files
│   ├── flows/          # Flow definitions
│   ├── apps/           # App configurations
│   └── logs/           # Execution logs
├── worker-data/
│   ├── cache/          # Language runtime cache
│   ├── tmp/            # Temporary execution files
│   └── pip/            # Python packages
└── postgres-data/      # Database files
```

### Data Persistence

- **Scripts/Flows**: Stored in PostgreSQL, cached on disk
- **Large Files**: S3-compatible storage (optional)
- **Logs**: Rotating files with configurable retention
- **Worker State**: Ephemeral, recreated on restart

## Execution Model

### Job Lifecycle

1. **Submission**: Client submits job via API
2. **Queuing**: Job added to Redis queue
3. **Assignment**: Worker picks up job based on tags/availability
4. **Execution**: Worker runs code in isolated environment
5. **Result Storage**: Output saved to PostgreSQL
6. **Notification**: Client notified via WebSocket

### Worker Types

#### Standard Workers
- Execute general scripts and flows
- Support all languages
- Default resource allocation

#### Tagged Workers
- Specialized for specific tasks
- Examples: `gpu`, `heavy`, `python-only`
- Custom resource limits

#### Dedicated Workers
- Assigned to specific workspaces
- Isolated execution environment
- Enhanced security

## Security Architecture

### Isolation Levels

1. **Process Isolation**
   - Each job runs in separate process
   - Resource limits enforced
   - Clean environment per execution

2. **Container Isolation**
   - Workers run in Docker containers
   - Network policies applied
   - Volume mounting restricted

3. **Workspace Isolation**
   - Logical separation of resources
   - Role-based access control
   - Audit logging

### Authentication Flow

```
Client ──────► API Server
   │               │
   │  JWT Token    │
   │◄──────────────┤
   │               │
   │  Authorized   │
   │  Request      │
   │──────────────►│
                   │
                   ▼
              PostgreSQL
```

## Performance Characteristics

### Scalability

- **Horizontal**: Add more workers
- **Vertical**: Increase worker resources
- **Queue-based**: Handles burst loads

### Bottlenecks

1. **Database**: Connection pooling critical
2. **Redis**: Memory usage with large queues
3. **Workers**: CPU/memory for compute tasks

### Optimization Points

- Worker count vs resource allocation
- Database connection pooling
- Redis memory management
- Log rotation and cleanup

## High Availability

### Component Redundancy

```yaml
# HA Configuration
Server:   Active-Passive with health checks
Workers:  N+1 redundancy
Database: Primary-Replica replication
Redis:    Sentinel for failover
```

### Failure Modes

1. **Worker Failure**: Jobs redistributed
2. **Server Failure**: Standby promotion
3. **Database Failure**: Read replica promotion
4. **Network Partition**: Queue-based recovery

## Integration Points

### Incoming

- REST API (HTTP/HTTPS)
- WebSocket (Real-time updates)
- Webhooks (External triggers)
- CLI tools

### Outgoing

- HTTP requests from scripts
- Database connections
- S3-compatible storage
- Email/Slack notifications

## Monitoring Points

### Key Metrics

```bash
# Server metrics
- API response time
- Active connections
- Queue depth

# Worker metrics  
- Job execution time
- Success/failure rate
- Resource utilization

# System metrics
- Database connections
- Redis memory usage
- Disk space
```

### Health Checks

- `/health` - Overall system health
- `/metrics` - Prometheus metrics
- Worker heartbeats
- Database connectivity

## Next Steps

- [Configure your installation](CONFIGURATION.md)
- [Understand core concepts](CONCEPTS.md)
- [Learn about operations](OPERATIONS.md)