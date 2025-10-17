# Neo4j Graph Database Resource

Neo4j is a native property graph database providing powerful graph queries and analysis capabilities for knowledge graphs, recommendations, and dependency analysis.

## Quick Start

```bash
# Install and start Neo4j
vrooli resource neo4j manage install
vrooli resource neo4j manage start

# Check status
vrooli resource neo4j status

# Run Cypher queries
vrooli resource neo4j content query "MATCH (n) RETURN count(n)"

# Performance monitoring
vrooli resource neo4j content metrics
vrooli resource neo4j content monitor "MATCH (n) RETURN n LIMIT 10"

# Import CSV data
vrooli resource neo4j content import-csv /path/to/data.csv \
  "LOAD CSV WITH HEADERS FROM 'file:///data.csv' AS row CREATE (n:Node) SET n = row"

# Backup and restore
vrooli resource neo4j content backup                          # Default filename
vrooli resource neo4j content backup --output /path/to/backup # Custom path
vrooli resource neo4j content restore <backup-file>
```

## Features

- **Native Graph Storage**: Optimized for storing and traversing relationships
- **Cypher Query Language**: Powerful declarative graph query language
- **ACID Transactions**: Full transaction support with rollback
- **Real-time Analysis**: Millisecond response for complex graph queries
- **Flexible Schema**: Schema-optional with constraints when needed
- **Performance Monitoring**: Track query performance and database metrics
- **Backup/Restore**: Full database backup and restoration capabilities
- **CSV Import**: Bulk import data from CSV files with custom Cypher queries
- **v2.0 Contract Compliant**: Follows Vrooli universal resource standards

## Documentation

- [Installation Guide](docs/installation.md)
- [Cypher Query Guide](docs/cypher.md)
- [Injection Guide](docs/injection.md)
- [Best Practices](docs/best-practices.md)

## Use Cases

- Knowledge graphs and semantic networks
- Recommendation engines
- Dependency and impact analysis
- Fraud detection and anomaly detection
- Network and IT operations
- Social network analysis

## Default Configuration

- **Port**: 7474 (HTTP), 7687 (Bolt)
- **Default Database**: neo4j
- **Authentication**: See [Authentication Setup](#authentication-setup) below
- **Memory**: 512MB heap (adjustable)
- **APOC Plugin**: Optional (install with `vrooli resource neo4j manage install-apoc`)

## Authentication Setup

### Development Mode (Default)
By default, Neo4j runs with authentication disabled (`NEO4J_AUTH="none"`) for ease of development:
```bash
# Start with no authentication (default)
vrooli resource neo4j manage start
```

### Production Mode
For production environments, **always set authentication**:
```bash
# Set authentication before starting
export NEO4J_AUTH="neo4j/YourStrongPassword123!"
vrooli resource neo4j manage start

# Or set in your environment file
echo 'NEO4J_AUTH="neo4j/YourStrongPassword123!"' >> ~/.bashrc
```

### Security Best Practices
- **Never use default passwords** in production
- **Use strong passwords** with mixed case, numbers, and special characters
- **Rotate passwords regularly** according to your security policy
- **Consider secrets management** tools like HashiCorp Vault for production
- **Restrict network access** using firewalls or Docker network isolation

### Checking Current Authentication
```bash
# View current authentication configuration
vrooli resource neo4j credentials
```

## Graph Algorithms and APOC

Neo4j supports advanced graph algorithms through the APOC (Awesome Procedures on Cypher) plugin:

### Installing APOC
```bash
# Install APOC plugin for enhanced functionality
vrooli resource neo4j manage install-apoc

# Restart Neo4j to activate
vrooli resource neo4j manage restart
```

### Available Algorithms

With APOC Core (Community Edition):
- **Shortest Path**: `vrooli resource neo4j content shortest-path --from 1 --to 5`
  - Uses `apoc.algo.dijkstra` for weighted paths
  - Uses `apoc.algo.aStar` for heuristic-based pathfinding
- **Centrality Metrics**: `vrooli resource neo4j content centrality --type degree`
  - Degree centrality (native Cypher implementation)
  - Betweenness and closeness (approximations)
- **PageRank** (Fallback): `vrooli resource neo4j content pagerank --node-label Person`
  - Note: True PageRank requires GDS (Graph Data Science) library
  - Falls back to degree centrality as approximation
- **Community Detection**: `vrooli resource neo4j content communities --algorithm louvain`
  - Basic connected components detection (native Cypher)
  - Advanced algorithms require GDS library
- **Similarity**: `vrooli resource neo4j content similarity --node-label Product`
  - Jaccard similarity (native Cypher implementation)

### Benefits of APOC
- Enhanced import/export capabilities (JSON, GraphML, etc.)
- Path-finding algorithms (Dijkstra, A*, all simple paths)
- Data integration utilities
- Atomic operations for thread-safe updates
- Collection and conversion utilities

### Note on Graph Algorithms
The Community Edition includes APOC Core which provides essential procedures. For advanced graph algorithms like PageRank, Louvain community detection, and Graph Neural Networks, you would need:
1. **Neo4j Enterprise Edition** + **Graph Data Science (GDS) Library**
2. Our implementation provides fallback algorithms using native Cypher for basic functionality

**Note**: Basic algorithm commands work without APOC but provide limited functionality. Install APOC for full feature access.

## Production Deployment Examples

### Docker Compose with SSL/TLS
```yaml
version: '3.8'
services:
  neo4j:
    image: neo4j:5.15.0
    container_name: neo4j-prod
    environment:
      NEO4J_AUTH: "${NEO4J_AUTH}"  # Set via .env file
      NEO4J_dbms_memory_heap_initial__size: "2G"
      NEO4J_dbms_memory_heap_max__size: "4G"
      NEO4J_dbms_memory_pagecache_size: "2G"
      NEO4J_dbms_connector_https_enabled: "true"
      NEO4J_dbms_ssl_policy_https_enabled: "true"
      NEO4J_dbms_ssl_policy_https_base__directory: "/ssl"
      NEO4J_dbms_ssl_policy_https_private__key: "private.key"
      NEO4J_dbms_ssl_policy_https_public__certificate: "public.crt"
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
      - ./ssl:/ssl:ro
    ports:
      - "7473:7473"  # HTTPS
      - "7687:7687"  # Bolt
    networks:
      - app_network
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "curl -f https://localhost:7473 || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5
```

### Kubernetes Deployment with Secrets
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: neo4j-auth
type: Opaque
data:
  username: bmVvNGo=  # base64 encoded "neo4j"
  password: <your-base64-encoded-password>
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: neo4j-config
data:
  NEO4J_dbms_memory_heap_initial__size: "2G"
  NEO4J_dbms_memory_heap_max__size: "4G"
  NEO4J_dbms_memory_pagecache_size: "2G"
  NEO4J_dbms_default__listen__address: "0.0.0.0"
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: neo4j
spec:
  serviceName: neo4j
  replicas: 1
  selector:
    matchLabels:
      app: neo4j
  template:
    metadata:
      labels:
        app: neo4j
    spec:
      containers:
      - name: neo4j
        image: neo4j:5.15.0
        env:
        - name: NEO4J_AUTH
          value: "$(NEO4J_USERNAME)/$(NEO4J_PASSWORD)"
        - name: NEO4J_USERNAME
          valueFrom:
            secretKeyRef:
              name: neo4j-auth
              key: username
        - name: NEO4J_PASSWORD
          valueFrom:
            secretKeyRef:
              name: neo4j-auth
              key: password
        envFrom:
        - configMapRef:
            name: neo4j-config
        ports:
        - containerPort: 7474
          name: http
        - containerPort: 7687
          name: bolt
        volumeMounts:
        - name: data
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /
            port: 7474
          initialDelaySeconds: 60
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 7474
          initialDelaySeconds: 30
          periodSeconds: 5
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

### Production Environment Variables
```bash
# .env file for production
NEO4J_AUTH="neo4j/$(openssl rand -base64 32)"
NEO4J_dbms_memory_heap_initial__size="4G"
NEO4J_dbms_memory_heap_max__size="8G"
NEO4J_dbms_memory_pagecache_size="4G"
NEO4J_dbms_logs_query_enabled="true"
NEO4J_dbms_logs_query_threshold="500ms"
NEO4J_dbms_logs_query_rotation_size="100M"
NEO4J_dbms_logs_query_rotation_keep__number="10"
```