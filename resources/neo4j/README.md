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

# Backup and restore
vrooli resource neo4j content backup
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
- **Authentication**: neo4j/VrooliNeo4j2024!
- **Memory**: 512MB heap (adjustable)