# Chart Generator 📊

Professional data visualization generation with customizable styling, supporting all major chart types. This scenario becomes the foundational charting engine that every business report, research analysis, financial dashboard, and data presentation scenario can leverage programmatically.

## 🎯 Purpose

The Chart Generator is a **foundational capability** that transforms Vrooli into a comprehensive business intelligence platform. Every scenario that generates reports, analyzes data, or presents information can now include professional-quality visualizations.

## 🚀 Key Features

### Core Chart Types
- **Bar Charts** - Perfect for comparing categories
- **Line Charts** - Ideal for trend analysis over time
- **Pie Charts** - Great for showing proportional data
- **Scatter Plots** - Excellent for correlation analysis
- **Area Charts** - Enhanced trend visualization with volume
- **Gantt Charts** - Project timeline visualization
- **Heatmaps** - Intensity visualization across two dimensions
- **Treemaps** - Hierarchical data representation
- **Candlestick Charts** - Financial data with open/high/low/close values ✅

### Professional Styling
- **Professional** - Clean, corporate styling for business reports
- **Minimal** - Ultra-clean, distraction-free for technical documentation
- **Vibrant** - Bold, eye-catching colors for marketing presentations
- **Dark Professional** - Modern dark theme for dashboards
- **Corporate** - Conservative styling for formal documentation
- **Financial** - Red/green coding perfect for financial data
- **Custom Styles** - Build your own styles with live preview ✅

### Advanced Features (All Implemented ✅)
- **Custom Style Builder** - Create and preview custom styles with color palette management ✅
- **Chart Animation** - Enable interactive tooltips and smooth transitions ✅
- **Live Preview** - See changes instantly while customizing styles ✅
- **Color Palettes** - Pre-defined color schemes (Ocean, Sunset, Forest, Corporate, Vibrant) ✅
- **Chart Composition** - Multiple charts in single canvas with grid/horizontal/vertical layouts ✅
- **Data Transformation** - Built-in aggregation, filtering, sorting, and grouping ✅
- **Template Library** - 15+ industry-specific templates for business, finance, healthcare, tech, marketing, and education ✅

### Export Formats
- **PNG** - High-quality raster images for documents
- **SVG** - Vector graphics for scalable, print-ready charts
- **PDF** - Direct PDF generation with embedded data tables and metadata ✅

## 🛠️ Quick Start

### 1. Start the Service
```bash
vrooli scenario run chart-generator
```

### 2. Generate Your First Chart
```bash
# Create sample data
echo '[{"x":"Q1","y":15000},{"x":"Q2","y":22000},{"x":"Q3","y":28000},{"x":"Q4","y":25000}]' > sales.json

# Generate a professional bar chart
chart-generator generate bar --data sales.json --style professional --format png,svg --title "Quarterly Sales"
```

### 3. Use the Web Interface
Open http://localhost:20301 for the style management interface

## 💻 CLI Usage

### Basic Commands
```bash
# Check service status
chart-generator status

# List available styles
chart-generator styles

# List chart templates
chart-generator templates

# Show help
chart-generator help
```

### Chart Generation Examples
```bash
# Bar chart from JSON file
chart-generator generate bar --data data.json --style professional --format png

# Line chart from CSV with custom styling
chart-generator generate line --data trends.csv --style vibrant --width 1200 --height 800

# Pie chart from stdin
echo '[{"x":"A","y":35},{"x":"B","y":28},{"x":"C","y":37}]' | chart-generator generate pie --data - --style minimal

# Multiple formats with title
chart-generator generate scatter --data analysis.json --format png,svg,pdf --title "Risk vs Return Analysis"

# Gantt chart for project planning
chart-generator generate gantt --data project-tasks.json --style professional --format png

# Candlestick chart for financial data
chart-generator generate candlestick --data stock-prices.json --style financial --format png,svg
```

## 🔌 API Integration

### Generate Chart via API
```bash
curl -X POST http://localhost:20300/api/v1/charts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "chart_type": "bar",
    "data": [{"x": "Q1", "y": 15000}, {"x": "Q2", "y": 22000}],
    "style": "professional",
    "export_formats": ["png", "svg"],
    "title": "Quarterly Performance"
  }'
```

### List Available Styles
```bash
curl http://localhost:20300/api/v1/styles
```

### Style Builder API

#### Get Color Palettes
```bash
curl http://localhost:20300/api/v1/styles/builder/palettes
```

#### Preview Custom Style
```bash
curl -X POST http://localhost:20300/api/v1/styles/builder/preview \
  -H "Content-Type: application/json" \
  -d '{
    "chart_type": "bar",
    "style": {
      "name": "Custom Ocean",
      "colors": ["#0369a1", "#0284c7", "#0ea5e9"],
      "font_family": "Arial",
      "font_size": 12,
      "background": "#ffffff",
      "grid_lines": true,
      "animation": true,
      "border_width": 2,
      "opacity": 0.8
    }
  }'
```

#### Save Custom Style
```bash
curl -X POST http://localhost:20300/api/v1/styles/builder/save \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Corporate Blue",
    "colors": ["#002366", "#0047AB", "#4169E1"],
    "font_family": "Arial",
    "font_size": 12,
    "background": "#ffffff",
    "grid_lines": false,
    "animation": false
  }'
```

### Composite Charts (NEW P1 Feature)
```bash
# Generate multiple charts in single canvas
curl -X POST http://localhost:20300/api/v1/charts/composite \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Executive Dashboard",
    "config": {
      "composition": {
        "layout": "grid",
        "grid": {"rows": 2, "columns": 2, "spacing": 10},
        "charts": [
          {"chart_type": "bar", "data": [{"x": "Q1", "y": 100}], "title": "Sales"},
          {"chart_type": "line", "data": [{"x": 1, "y": 10}], "title": "Trend"},
          {"chart_type": "pie", "data": [{"label": "A", "value": 60}], "title": "Share"},
          {"chart_type": "area", "data": [{"x": 1, "y": 5}], "title": "Volume"}
        ]
      }
    }
  }'
```

### Data Transformation (NEW P1 Feature)
```bash
# Transform data with aggregation, filtering, and sorting
curl -X POST http://localhost:20300/api/v1/data/transform \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"category": "A", "value": 100, "date": "2025-01"},
      {"category": "B", "value": 150, "date": "2025-01"},
      {"category": "A", "value": 200, "date": "2025-02"}
    ],
    "transform": {
      "filter": {"field": "value", "operator": "gt", "value": 120},
      "aggregate": {"method": "sum", "field": "value", "group_by": "category"},
      "sort": {"field": "value", "direction": "desc"}
    }
  }'
```

### Industry Templates (NEW P1 Feature)
```bash
# List all templates
curl http://localhost:20300/api/v1/templates

# Filter by industry
curl http://localhost:20300/api/v1/templates?industry=finance

# Get specific template
curl http://localhost:20300/api/v1/templates/stock-performance
```

## 🎨 Style Management

### Web Interface Features
- **Style Gallery** - Browse all available chart styles with previews
- **Custom Style Creator** - Build your own color palettes and themes
- **Live Preview** - See changes in real-time with sample data
- **Template Management** - Save and reuse common chart configurations

### Creating Custom Styles
1. Open the style management UI at http://localhost:20301
2. Click "Create Custom Style"
3. Configure colors, typography, and spacing
4. Preview with sample data
5. Save for future use

## 🔄 Cross-Scenario Integration

The Chart Generator is designed to be used by other scenarios:

### Business Reports
```bash
# From business-reports scenario
chart-generator generate bar --data quarterly-metrics.json --style corporate
```

### Financial Analysis
```bash
# From financial-analyzer scenario
chart-generator generate line --data stock-performance.json --style financial
```

### Research Papers
```bash
# From research-assistant scenario
chart-generator generate scatter --data research-data.json --style minimal
```

## 📊 Data Format Requirements

### JSON Format
```json
[
  {"x": "Category A", "y": 100},
  {"x": "Category B", "y": 150},
  {"x": "Category C", "y": 120}
]
```

### CSV Format
```csv
x,y
Category A,100
Category B,150
Category C,120
```

### Advanced Data (with additional properties)
```json
[
  {"x": "Q1", "y": 15000, "label": "First Quarter", "color": "#2563eb"},
  {"x": "Q2", "y": 22000, "label": "Second Quarter", "color": "#10b981"}
]
```

### Specialized Chart Data Formats

#### Gantt Chart
```json
[
  {"task": "Design Phase", "start": "2024-01-01", "end": "2024-01-15"},
  {"task": "Development", "start": "2024-01-10", "end": "2024-02-15"}
]
```

#### Candlestick Chart (Financial)
```json
[
  {"date": "2024-01-01", "open": 100, "high": 110, "low": 95, "close": 105},
  {"date": "2024-01-02", "open": 105, "high": 115, "low": 100, "close": 112}
]
```

#### Heatmap
```json
[
  {"x": "Monday", "y": "Morning", "value": 10},
  {"x": "Monday", "y": "Afternoon", "value": 20}
]
```

## 🆕 New Features (v1.0 - 2025-09-27)

### PDF Export
Generate PDF reports with embedded charts and data:
```bash
# Generate a PDF report with chart and data table
curl -X POST http://localhost:20300/api/v1/charts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "chart_type": "bar",
    "title": "Q4 2024 Sales Report",
    "data": [{"x": "Oct", "y": 45000}, {"x": "Nov", "y": 52000}, {"x": "Dec", "y": 61000}],
    "export_formats": ["pdf"]
  }'
```

### Comprehensive Testing
Run the full test suite to validate all features:
```bash
# Run comprehensive tests (15 test cases)
./test/test-chart-generation.sh
```

### Enhanced Chart Validation
- Improved data validation for all chart types
- Better error messages for malformed data
- Support for multiple field name conventions (name/label/x for categories)

## 🏗️ Architecture

### Components
- **Go API Server** - Fast, concurrent chart generation service
- **PostgreSQL Database** - Stores styles, templates, and generation history
- **N8N Workflows** - Orchestrates chart generation pipeline
- **Web UI** - Style management and preview interface
- **CLI Tool** - Command-line interface for integration

### Storage
- **Chart Styles** - Reusable styling configurations
- **Chart Templates** - Pre-configured chart setups
- **Generation History** - Performance metrics and usage analytics
- **Temporary Files** - Auto-cleanup after 7 days

## 🎯 Use Cases

### Business Intelligence
- Executive dashboards with real-time KPIs
- Quarterly performance reports with consistent styling
- Financial presentations with professional appearance

### Research & Academia
- Scientific paper visualizations with publication quality
- Data analysis reports with minimal, clean styling
- Academic presentations with clear, readable charts

### Marketing & Creative
- Campaign performance dashboards with vibrant styling
- Creative presentations that need visual impact
- Brand-consistent reporting across all materials

## 🔧 Configuration

### Environment Variables
```bash
# API Configuration
CHART_API_PORT=20300        # Chart generation API port
CHART_UI_PORT=20301         # Style management UI port

# Database Configuration  
POSTGRES_HOST=localhost     # PostgreSQL host
POSTGRES_PORT=5432         # PostgreSQL port
POSTGRES_DB=chart_generator # Database name
```

### Service Dependencies
- **PostgreSQL** - Required for data persistence
- **N8N** - Required for workflow orchestration
- **MinIO** - Optional for file storage
- **Redis** - Optional for caching

## 📈 Performance

### Benchmarks
- **Generation Time** - < 2 seconds for 1000+ data points
- **Throughput** - 50 charts/minute per instance
- **Memory Usage** - < 512MB per generation process
- **Export Quality** - Vector-perfect SVG, 300+ DPI PNG

### Scalability
- Horizontal scaling via multiple API instances
- Database connection pooling for concurrent requests
- Automatic cleanup of temporary files
- Performance monitoring and analytics

## 🔒 Security & Privacy

### Data Protection
- Generated charts expire automatically after 7 days
- No persistent storage of sensitive data beyond expiration
- Secure API endpoints with proper validation
- Audit trail of all generation activities

### Access Control
- Style creation requires proper authentication
- Template sharing controls for organizational use
- Rate limiting to prevent abuse
- Comprehensive logging for security monitoring

## 📚 Documentation

- **[API Documentation](docs/api.md)** - Complete REST API reference
- **[Style Guide](docs/styles.md)** - Creating and managing chart styles
- **[Integration Guide](docs/integration.md)** - Using charts in other scenarios
- **[Performance Tuning](docs/performance.md)** - Optimizing for high-volume usage

---

**Chart Generator** - Transforming data into insight, one chart at a time. 📊✨
## API Endpoints

### Search Endpoint (Enhanced)
```bash
# Advanced search with filters
curl -X POST http://localhost:PORT/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "artificial intelligence research",
    "limit": 10,
    "language": "en",
    "safe_search": 1,
    "file_type": "pdf",
    "site": "arxiv.org",
    "exclude_sites": ["wikipedia.org"],
    "time_range": "month",
    "sort_by": "date",
    "region": "us",
    "min_date": "2024-01-01",
    "engines": ["google", "bing", "duckduckgo"]
  }'
```

### Report Management
```bash
# Create a new research report
curl -X POST http://localhost:PORT/api/reports \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "Quantum computing advances",
    "depth": "deep",
    "target_length": 10,
    "language": "en",
    "tags": ["quantum", "computing", "research"],
    "category": "technology"
  }'

# Get all reports
curl http://localhost:PORT/api/reports

# Get specific report
curl http://localhost:PORT/api/reports/{id}
```

### Dashboard Statistics
```bash
curl http://localhost:PORT/api/dashboard/stats
```

## CLI Usage
```bash
# Check status
research-assistant status

# Get help
research-assistant help

# Note: Additional CLI commands for report creation and chat are planned
```

## Quick Start
1. Ensure required resources are running:
   ```bash
   vrooli resource status ollama
   vrooli resource status qdrant
   vrooli resource status searxng
   ```

2. Start the scenario:
   ```bash
   vrooli scenario run research-assistant
   ```

3. Access the services:
   - API: http://localhost:PORT (check output for actual port)
   - UI: http://localhost:PORT (check output for actual port)

## Known Limitations
- Windmill resource integration pending (optional)
- n8n workflows require manual import
- Some P1 features in development (contradiction detection, report templates)

## Recent Updates (2025-09-30)
- Implemented advanced search filters with comprehensive filtering options
- Fixed SearXNG integration for reliable multi-source searching
- Enhanced CLI port detection for better reliability
- Added sorting capabilities for search results
