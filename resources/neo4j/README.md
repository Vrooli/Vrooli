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
- **Authentication**: neo4j/VrooliNeo4j2024!
- **Memory**: 512MB heap (adjustable)
- **APOC Plugin**: Optional (install with `vrooli resource neo4j manage install-apoc`)

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