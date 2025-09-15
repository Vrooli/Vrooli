# Neo4j Resource - Known Issues and Solutions

## Performance Monitoring Limitations

### Issue
Some DBMS procedures like `dbms.listTransactions` are not available in Neo4j Community Edition.

### Impact
- Performance metrics are limited to basic stats (memory, database size)
- Cannot retrieve active transaction details

### Workaround
- Use alternative monitoring approaches through logs
- Consider upgrading to Enterprise Edition for full monitoring capabilities

## APOC Plugin

### Issue
APOC (Awesome Procedures on Cypher) plugin is referenced but may not be installed by default.

### Impact
- Some advanced features like JSON import/export may not work
- Graph algorithms fallback to basic implementations

### Solution
The resource includes fallback implementations for critical features. To enable full APOC functionality:
```bash
docker exec vrooli-neo4j bash -c "cd /var/lib/neo4j/plugins && wget https://github.com/neo4j-contrib/neo4j-apoc-procedures/releases/download/5.15.0/apoc-5.15.0-core.jar"
docker restart vrooli-neo4j
```

## Memory Usage

### Issue
Neo4j can consume significant memory, especially with large graphs.

### Current Settings
- Default heap size: 512MB minimum
- May need adjustment for production workloads

### Solution
Adjust memory settings in container environment variables if needed:
- NEO4J_server_memory_heap_initial__size
- NEO4J_server_memory_heap_max__size
- NEO4J_server_memory_pagecache_size

## Authentication

### Current Implementation
Uses hardcoded password "VrooliNeo4j2024!" for simplicity.

### Security Note
For production deployments, credentials should be:
- Stored in environment variables
- Rotated regularly
- Use stronger passwords