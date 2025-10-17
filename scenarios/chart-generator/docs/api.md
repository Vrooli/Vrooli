# Chart Generator API Documentation

## Table of Contents
- [Overview](#overview)
- [Base URL](#base-url)
- [Authentication](#authentication)
- [Core Endpoints](#core-endpoints)
- [Advanced Features](#advanced-features)
- [Templates & Styles](#templates--styles)
- [Error Handling](#error-handling)
- [Examples](#examples)

## Overview

The Chart Generator API provides professional-grade data visualization generation with customizable styling, supporting all major chart types. This API serves as the foundational charting engine for business reports, research analysis, financial dashboards, and data presentations.

## Base URL

```
http://localhost:19095
```

The API automatically discovers its port through the Vrooli lifecycle system. You can verify the port using:
```bash
vrooli scenario port chart-generator | grep API_PORT
```

## Authentication

Currently, the API does not require authentication for local development. In production deployments, authentication should be implemented based on your security requirements.

## Core Endpoints

### Health Check

Check if the API is healthy and responding.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-27T18:45:19.852137648-04:00",
  "version": "1.0.0",
  "service": "chart-generator-api"
}
```

### Generate Chart

Generate a chart from provided data and configuration.

**Endpoint:** `POST /api/v1/charts/generate`

**Request Body:**
```json
{
  "chart_type": "bar|line|pie|scatter|area|gantt|heatmap|treemap|candlestick",
  "title": "Optional chart title",
  "data": [
    {"x": "Label1", "y": 100},
    {"x": "Label2", "y": 200}
  ],
  "style_id": "professional|minimal|vibrant|dark|corporate",
  "export_formats": ["png", "svg", "pdf"],
  "config": {
    "width": 800,
    "height": 600,
    "show_legend": true,
    "show_grid": true,
    "animation": false
  }
}
```

**Response:**
```json
{
  "success": true,
  "chart_id": "chart_1759013145_abc123",
  "files": {
    "png": "/tmp/chart_1759013145_abc123_output/chart.png",
    "svg": "/tmp/chart_1759013145_abc123_output/chart.svg",
    "pdf": "/tmp/chart_1759013145_abc123_output/chart.pdf"
  },
  "metadata": {
    "generation_time_ms": 11,
    "data_point_count": 2,
    "style_applied": "professional",
    "dimensions": {
      "width": 800,
      "height": 600
    },
    "formats_generated": ["png", "svg", "pdf"],
    "created_at": "2025-09-27T18:45:45-04:00"
  }
}
```

### Generate Interactive Chart

Generate a chart with interactive features and animations.

**Endpoint:** `POST /api/v1/charts/interactive`

**Request Body:**
```json
{
  "chart_type": "bar|line|pie|scatter|area",
  "data": [{"x": "A", "y": 10}, {"x": "B", "y": 20}],
  "animation_config": {
    "enable_transitions": true,
    "enable_hover": true,
    "enable_tooltips": true,
    "enable_zoom": true,
    "enable_pan": true,
    "enable_selection": true
  }
}
```

**Response:**
```json
{
  "success": true,
  "chart_id": "chart_1759013175_5rlfjd",
  "files": {
    "png": "/tmp/chart_1759013175_5rlfjd_output/chart.png"
  },
  "metadata": {
    "animation_enabled": true,
    "interactive": true,
    "features": [
      "animation",
      "tooltips",
      "legend_interaction",
      "zoom",
      "pan",
      "data_zoom"
    ],
    "generation_time_ms": 6
  }
}
```

## Advanced Features

### Composite Charts

Create composite charts with multiple visualizations in a single canvas.

**Endpoint:** `POST /api/v1/charts/composite`

**Request Body:**
```json
{
  "chart_type": "composite",
  "config": {
    "composition": {
      "layout": "grid|horizontal|vertical",
      "charts": [
        {
          "chart_type": "bar",
          "data": [{"x": "A", "y": 10}],
          "position": {"row": 0, "col": 0}
        },
        {
          "chart_type": "line",
          "data": [{"x": "Q1", "y": 30}],
          "position": {"row": 0, "col": 1}
        }
      ]
    }
  }
}
```

### Data Transformation

Transform data before generating charts.

**Endpoint:** `POST /api/v1/data/transform`

**Request Body:**
```json
{
  "data": [
    {"x": "C", "y": 30},
    {"x": "A", "y": 10},
    {"x": "B", "y": 20}
  ],
  "transform": {
    "sort": {
      "field": "y",
      "direction": "asc|desc"
    },
    "filter": {
      "field": "y",
      "operator": "gt|gte|lt|lte|eq|neq",
      "value": 15
    }
  }
}
```

### Data Aggregation

Aggregate data for summarization.

**Endpoint:** `POST /api/v1/data/aggregate`

**Request Body:**
```json
{
  "data": [
    {"category": "A", "value": 10},
    {"category": "A", "value": 20},
    {"category": "B", "value": 15}
  ],
  "method": "sum|avg|min|max|count",
  "field": "value",
  "group_by": "category"
}
```

### Data Validation

Validate data before chart generation.

**Endpoint:** `POST /api/v1/data/validate`

**Request Body:**
```json
{
  "chart_type": "bar",
  "data": [{"x": "A", "y": 10}]
}
```

**Response:**
```json
{
  "valid": true,
  "errors": [],
  "warnings": [],
  "data_points": 1,
  "chart_type": "bar",
  "message": "Data validation successful"
}
```

## Templates & Styles

### List Styles

Get all available chart styles.

**Endpoint:** `GET /api/v1/styles`

**Response:**
```json
{
  "count": 5,
  "styles": [
    {
      "id": "professional",
      "name": "Professional",
      "category": "business",
      "description": "Clean, corporate styling",
      "is_default": true
    },
    {
      "id": "minimal",
      "name": "Minimal",
      "category": "clean",
      "description": "Ultra-clean design",
      "is_default": false
    }
  ]
}
```

### List Templates

Get all industry-specific chart templates.

**Endpoint:** `GET /api/v1/templates`

**Response:**
```json
{
  "count": 15,
  "templates": [
    {
      "id": "quarterly-sales",
      "name": "Quarterly Sales Performance",
      "category": "business",
      "industry": "retail",
      "chart_type": "bar",
      "description": "Standard quarterly sales chart"
    },
    {
      "id": "stock-performance",
      "name": "Stock Performance Dashboard",
      "category": "financial",
      "industry": "finance",
      "chart_type": "candlestick",
      "description": "OHLC stock data visualization"
    }
  ]
}
```

### Style Builder Preview

Preview a custom style before saving.

**Endpoint:** `POST /api/v1/styles/builder/preview`

**Request Body:**
```json
{
  "chart_type": "bar",
  "style": {
    "name": "Custom Style",
    "color_palette": ["#FF5733", "#33FF57", "#3357FF"],
    "font_family": "Arial",
    "background_color": "#F0F0F0"
  }
}
```

### Get Color Palettes

Get available color palettes for style building.

**Endpoint:** `GET /api/v1/styles/builder/palettes`

**Response:**
```json
{
  "palettes": [
    {
      "id": "ocean",
      "name": "Ocean Blues",
      "colors": ["#003f5c", "#2f4b7c", "#665191", "#a05195", "#d45087"]
    },
    {
      "id": "sunset",
      "name": "Sunset Warm",
      "colors": ["#f95d6a", "#ff7c43", "#ffa600", "#ffd700", "#ffed6f"]
    }
  ]
}
```

## Error Handling

All endpoints follow a consistent error response format:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_CHART_TYPE",
    "message": "The specified chart type 'invalid' is not supported",
    "details": {
      "supported_types": ["bar", "line", "pie", "scatter", "area"]
    }
  }
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `INVALID_CHART_TYPE` | Chart type not supported |
| `EMPTY_DATA` | No data provided for chart generation |
| `INVALID_DATA_FORMAT` | Data format doesn't match chart requirements |
| `EXPORT_FORMAT_ERROR` | Requested export format not supported |
| `GENERATION_FAILED` | Chart generation process failed |
| `STYLE_NOT_FOUND` | Specified style ID doesn't exist |
| `TEMPLATE_NOT_FOUND` | Specified template ID doesn't exist |

## Examples

### Basic Bar Chart

```bash
curl -X POST http://localhost:19095/api/v1/charts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "chart_type": "bar",
    "title": "Q4 Sales Report",
    "data": [
      {"x": "Product A", "y": 1000},
      {"x": "Product B", "y": 1500},
      {"x": "Product C", "y": 800}
    ],
    "export_formats": ["png", "svg"]
  }'
```

### Financial Candlestick Chart

```bash
curl -X POST http://localhost:19095/api/v1/charts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "chart_type": "candlestick",
    "title": "AAPL Stock Performance",
    "data": [
      {"date": "2024-01-01", "open": 100, "high": 110, "low": 95, "close": 105},
      {"date": "2024-01-02", "open": 105, "high": 115, "low": 100, "close": 112}
    ],
    "style_id": "professional",
    "export_formats": ["pdf"]
  }'
```

### Interactive Line Chart with Animations

```bash
curl -X POST http://localhost:19095/api/v1/charts/interactive \
  -H "Content-Type: application/json" \
  -d '{
    "chart_type": "line",
    "data": [
      {"x": "Jan", "y": 100},
      {"x": "Feb", "y": 150},
      {"x": "Mar", "y": 130}
    ],
    "animation_config": {
      "enable_transitions": true,
      "enable_hover": true,
      "enable_tooltips": true,
      "enable_zoom": true
    }
  }'
```

### Data Transformation Example

```bash
# First transform the data
TRANSFORMED_DATA=$(curl -X POST http://localhost:19095/api/v1/data/transform \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"x": "Item3", "y": 30},
      {"x": "Item1", "y": 10},
      {"x": "Item2", "y": 20}
    ],
    "transform": {
      "sort": {"field": "y", "direction": "desc"}
    }
  }' | jq -r '.data')

# Then generate chart with transformed data
curl -X POST http://localhost:19095/api/v1/charts/generate \
  -H "Content-Type: application/json" \
  -d "{
    \"chart_type\": \"bar\",
    \"data\": $TRANSFORMED_DATA,
    \"export_formats\": [\"png\"]
  }"
```

## Rate Limits

In production deployments, consider implementing rate limiting:
- Default: 100 requests per minute per IP
- Bulk generation: 10 requests per minute
- Large datasets (>1000 points): 5 requests per minute

## Performance Guidelines

- Charts with <100 data points: <50ms generation time
- Charts with 100-1000 data points: <500ms generation time
- Charts with 1000+ data points: <2000ms generation time
- Composite charts: Add 100ms per sub-chart
- PDF export: Add 200-500ms to generation time

## Version History

- **v1.0.0** (2025-09-27): Initial release with core chart types, templates, and interactive features