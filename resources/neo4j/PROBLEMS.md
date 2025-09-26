# Neo4j Resource - Known Issues and Solutions

## Code Quality Status (2025-09-26)

### All Shellcheck Issues Resolved
All shellcheck warnings have been addressed:
- No SC2086 warnings (unquoted variables)
- No SC2181 warnings (indirect exit code checks)
- No SC2006 warnings (backtick usage)
- No SC2155 warnings (combined declaration/assignment)
- No SC2154 warnings (undefined variables)

The codebase now follows bash best practices and is shellcheck-clean.

## Graph Algorithms Limitations

### Issue
Graph algorithms (PageRank, shortest path, etc.) require APOC plugin which is not installed by default.

### Solution
Install APOC plugin using the provided command:
```bash
vrooli resource neo4j manage install-apoc
vrooli resource neo4j manage restart
```

### Alternative Workarounds
- Use basic Cypher queries for graph traversal and analysis
- Consider using application-level graph algorithm implementations

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
APOC (Awesome Procedures on Cypher) plugin provides advanced functionality but is not installed by default.

### Solution
The resource now includes a command to install APOC:
```bash
# Install APOC plugin
vrooli resource neo4j manage install-apoc

# Restart to activate
vrooli resource neo4j manage restart
```

### Features Enabled by APOC
- Advanced import/export (JSON, GraphML, CSV)
- Full graph algorithm support (PageRank, community detection, etc.)
- Data integration utilities
- Temporal and spatial functions

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
Uses environment variable NEO4J_AUTH for authentication configuration.
- Defaults to "none" for development environments (no authentication)
- Can be set to "neo4j/yourpassword" for production use

### Security Note
For production deployments:
- Always set NEO4J_AUTH environment variable explicitly
- Use strong passwords and rotate regularly
- Consider using secrets management tools like Vault

## Backup/Restore Improvements (FIXED)

### Previous Issue
Backup exports were creating empty JSON files when authentication was set to "none".

### Solution Implemented
- Fixed backup implementation to properly export data in no-auth mode
- Improved JSON export format to correctly capture nodes and relationships
- Added fallback to basic export when APOC is not available

### Current Status
- Backups now work correctly with or without authentication
- Both APOC and basic export methods are supported
- JSON format includes nodes, relationships, and metadata