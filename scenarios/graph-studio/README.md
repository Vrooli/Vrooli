# Graph Studio

**Unified platform for creating, validating, and converting all forms of graph-based visualizations and data structures**

## 🎯 Purpose

Graph Studio consolidates the functionality of 10+ different diagramming and visualization tools into a single, extensible platform. It provides a plugin-based architecture that supports everything from simple mind maps to complex semantic knowledge graphs, making it the foundation for visual thinking across all Vrooli scenarios.

## 🚀 Quick Start

```bash
# Install the scenario
cd scenarios/graph-studio
./cli/install.sh

# Start the services
vrooli scenario run graph-studio

# Create your first graph
graph-studio create "My Project Plan" mind-maps

# List available plugins
graph-studio plugins

# Convert between formats
graph-studio convert <graph-id> mermaid
```

## 🔌 Available Plugins

### Visualization
- **Mind Maps** - Hierarchical thought organization (FreeMind, XMind, Mermaid)
- **Network Graphs** - Relationship modeling (Cytoscape, GraphML, GEXF, D3.js)
- **Mermaid** - Text-based diagramming

### Process Modeling
- **BPMN 2.0** - Business Process Model and Notation
- **DMN** - Decision Model and Notation (planned)
- **CMMN** - Case Management (planned)

### Architecture
- **UML** - Software design diagrams (planned)
- **ERD** - Database entity relationships (planned)
- **SysML** - Systems engineering (planned)
- **ArchiMate** - Enterprise architecture (planned)

### Semantic Web
- **RDF** - Resource Description Framework (planned)
- **OWL** - Web Ontology Language (planned)
- **JSON-LD** - Linked data (planned)

## 📚 Core Features

### Plugin Architecture
- Dynamic plugin loading
- Format conversion between compatible types
- Extensible validation system
- Custom rendering engines

### Data Management
- PostgreSQL storage with JSONB for flexibility
- Version control for all graphs
- Relationship tracking between graphs
- Template library for quick starts

### Integration
- RESTful API for programmatic access
- CLI for automation and scripting
- Event system for real-time updates
- Cross-scenario integration capabilities

## 🛠️ API Usage

### Create a Graph
```bash
# Get the API port dynamically
API_PORT=$(vrooli scenario port graph-studio API_PORT)
curl -X POST http://localhost:$API_PORT/api/v1/graphs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "System Architecture",
    "type": "network-graphs",
    "data": {
      "nodes": [{"id": "api", "label": "API Server"}],
      "edges": []
    }
  }'
```

### Validate a Graph
```bash
API_PORT=$(vrooli scenario port graph-studio API_PORT)
curl -X POST http://localhost:$API_PORT/api/v1/graphs/{id}/validate
```

### Convert Format
```bash
API_PORT=$(vrooli scenario port graph-studio API_PORT)
curl -X POST http://localhost:$API_PORT/api/v1/graphs/{id}/convert \
  -H "Content-Type: application/json" \
  -d '{"target_format": "mermaid"}'
```

### Render Graph
```bash
API_PORT=$(vrooli scenario port graph-studio API_PORT)
curl -X POST http://localhost:$API_PORT/api/v1/graphs/{id}/render \
  -H "Content-Type: application/json" \
  -d '{"format": "svg"}'
```

## 🎨 UI Features

The Graph Studio dashboard provides:
- **Plugin Gallery** - Visual cards for each graph type
- **Smart Creation** - AI-suggested graph types based on description
- **Live Preview** - Real-time rendering as you edit
- **Conversion Matrix** - See which formats can convert to others
- **Template Library** - Start from pre-built templates

## 🔄 Integration with Other Scenarios

Graph Studio serves as a foundational capability for:

- **research-assistant** - Visualize research connections as knowledge graphs
- **product-manager** - Create and manage product roadmaps
- **system-monitor** - Visualize system dependencies
- **stream-of-consciousness-analyzer** - Convert thoughts to mind maps
- **process-optimizer** - Analyze and improve BPMN workflows

## 📊 Business Value

- **Eliminates** need for 10+ separate diagramming tools
- **Saves** 500+ hours/year on format conversions
- **Revenue potential** $30K-$80K per enterprise deployment
- **Reusability** 10/10 - Every scenario can leverage graphs

## 🔧 Development

### Adding a New Plugin

1. Create plugin directory: `ui/src/plugins/my-plugin/`
2. Implement the IGraphPlugin interface
3. Register in `service.json`
4. Add validation and rendering logic

### Format Conversion

Conversions are handled through a compatibility matrix. To add a new conversion:

1. Define conversion rules in the plugin
2. Implement transform function
3. Register in conversion matrix
4. Add tests for data integrity

## 🧪 Testing

```bash
# Run all tests
./test.sh

# Test specific plugin
./test.sh --plugin mind-maps

# Validate all graphs
graph-studio validate --all
```

## 📖 Documentation

- [API Reference](docs/api.md)
- [Plugin Development Guide](docs/plugins.md)
- [Format Specifications](docs/formats.md)
- [Architecture Overview](docs/architecture.md)

## 🚨 Troubleshooting

### Common Issues

**Cannot connect to API**
- Ensure PostgreSQL is running: `vrooli resource postgres status`
- Check API logs: `vrooli scenario logs graph-studio`
- Verify scenario is running: `vrooli scenario status graph-studio`

**Plugin not loading**
- Verify plugin is enabled in `service.json`
- Check plugin directory structure
- Review API logs for loading errors

**Conversion failing**
- Ensure formats are compatible
- Check source graph is valid
- Review conversion matrix

## 🔮 Future Enhancements

- **Version 2.0**
  - Real-time collaboration
  - AI-powered auto-completion
  - SPARQL and GraphQL query interfaces
  - 3D graph visualizations

- **Long-term Vision**
  - 50+ supported graph formats
  - ML-based pattern recognition
  - Automatic graph generation from any data source
  - Become the "Figma for structured thinking"

## 📝 License

Part of the Vrooli ecosystem - see main project license.

---

**Graph Studio** - Where every idea finds its perfect visual form.