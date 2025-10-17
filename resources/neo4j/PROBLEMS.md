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

## Graph Algorithms Limitations (PARTIALLY RESOLVED)

### Current State
APOC Core is now installed and functional, providing:
- Path-finding algorithms (Dijkstra, A*)
- Atomic operations and collection utilities
- Enhanced import/export capabilities

### Remaining Limitations
Advanced algorithms (PageRank, Louvain community detection, Graph Neural Networks) require:
- Neo4j Enterprise Edition + Graph Data Science (GDS) Library
- The resource provides Cypher-based fallback implementations

### Solution
APOC Core is installed automatically:
```bash
vrooli resource neo4j manage install-apoc
vrooli resource neo4j manage restart
```

## Performance Monitoring Limitations

### Issue
Some DBMS procedures like `dbms.listTransactions` are not available in Neo4j Community Edition.

### Impact
- Performance metrics are limited to basic stats (memory, database size)
- Cannot retrieve active transaction details

### Workaround
- Use alternative monitoring approaches through logs
- Consider upgrading to Enterprise Edition for full monitoring capabilities

## APOC Plugin (RESOLVED - 2025-09-30)

### Status
APOC Core is now automatically installed and configured with proper file operation permissions.
- **Fixed**: APOC installation URL corrected from non-existent version "2025.09.0" to compatible "5.26.12"
- **Verified**: APOC loads correctly on restart and all functions work

### Configuration Applied
- `NEO4J_apoc_export_file_enabled=true` - Enables file exports
- `NEO4J_apoc_import_file_enabled=true` - Enables file imports
- Plugin loads automatically on container start

### Features Available with APOC Core
- ✅ Enhanced import/export (JSON, CSV)
- ✅ Path-finding algorithms (Dijkstra, A*, all simple paths)
- ✅ Atomic operations for thread-safe updates
- ✅ Collection and conversion utilities
- ⚠️ Limited graph algorithms (Enterprise + GDS required for advanced features)

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
Uses environment variable NEO4J_AUTH for authentication configuration:
- **Development Default**: `NEO4J_AUTH="none"` (no authentication for ease of development)
- **Production Requirement**: `NEO4J_AUTH="neo4j/YourStrongPassword"`

### Production Security Requirements
⚠️ **CRITICAL**: Never run with `NEO4J_AUTH="none"` in production!

**Required for Production:**
1. **Set Strong Authentication**:
   ```bash
   export NEO4J_AUTH="neo4j/YourStrongPassword123!"
   ```
2. **Password Requirements**:
   - Minimum 12 characters
   - Mixed case letters
   - Numbers and special characters
   - No dictionary words or patterns

3. **Security Best Practices**:
   - Rotate passwords every 30-90 days
   - Use different passwords per environment
   - Store credentials in secrets management (Vault, AWS Secrets Manager, etc.)
   - Restrict network access via firewall rules
   - Enable audit logging for compliance

### Development vs Production Configuration
```bash
# Development (local testing only)
export NEO4J_AUTH="none"  # OK for local development

# Staging/Production
export NEO4J_AUTH="neo4j/$(openssl rand -base64 20)"  # Generate strong password
```

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