# Neo4j Graph Database Resource

Neo4j is a native property graph database providing powerful graph queries and analysis capabilities for knowledge graphs, recommendations, and dependency analysis.

## Quick Start

```bash
# Install and start Neo4j
vrooli resource install neo4j
vrooli resource start neo4j

# Check status
vrooli resource neo4j status
resource-neo4j status

# Run Cypher queries
resource-neo4j query "MATCH (n) RETURN count(n)"

# Inject Cypher files
resource-neo4j inject path/to/queries.cypher
```

## Features

- **Native Graph Storage**: Optimized for storing and traversing relationships
- **Cypher Query Language**: Powerful declarative graph query language
- **ACID Transactions**: Full transaction support with rollback
- **Real-time Analysis**: Millisecond response for complex graph queries
- **Flexible Schema**: Schema-optional with constraints when needed

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