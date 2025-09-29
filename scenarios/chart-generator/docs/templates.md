# Chart Generator Template Guide

## Overview

The Chart Generator includes 15 pre-configured industry templates across 6 categories. Each template is optimized for specific use cases with appropriate styling, data formatting, and visualization choices.

## Template Categories

### 📊 Business Templates

Business templates focus on corporate reporting and operational metrics.

#### 1. Quarterly Sales Performance
**ID:** `quarterly-sales`  
**Chart Type:** Bar Chart  
**Industry:** Retail  
**Best For:** Comparing sales performance across quarters with year-over-year comparisons

**Usage:**
```bash
# CLI usage
chart-generator generate bar --template quarterly-sales --data sales.json

# API usage
curl -X POST http://localhost:19095/api/v1/charts/generate \
  -d '{
    "template_id": "quarterly-sales",
    "data": [
      {"quarter": "Q1", "current_year": 15000, "previous_year": 12000},
      {"quarter": "Q2", "current_year": 22000, "previous_year": 18000}
    ]
  }'
```

#### 2. Revenue Trend Analysis
**ID:** `revenue-trend`  
**Chart Type:** Line Chart  
**Industry:** SaaS  
**Best For:** Monthly revenue tracking with forecasting

**Usage:**
```bash
# Generate revenue trend with forecast
chart-generator generate line --template revenue-trend --data monthly-revenue.json

# Data format
[
  {"month": "Jan", "revenue": 45000, "forecast": 48000},
  {"month": "Feb", "revenue": 52000, "forecast": 55000}
]
```

#### 3. Market Share Distribution
**ID:** `market-share`  
**Chart Type:** Pie Chart  
**Industry:** Retail  
**Best For:** Visualizing competitive market positions

**Usage:**
```bash
# Show market share breakdown
chart-generator generate pie --template market-share --data market-data.json

# Data format
[
  {"company": "Your Company", "share": 35},
  {"company": "Competitor A", "share": 28},
  {"company": "Competitor B", "share": 22},
  {"company": "Others", "share": 15}
]
```

### 💰 Financial Templates

Financial templates provide specialized visualizations for financial data.

#### 4. Stock Performance Dashboard
**ID:** `stock-performance`  
**Chart Type:** Candlestick  
**Industry:** Finance  
**Best For:** OHLC (Open, High, Low, Close) stock data with volume indicators

**Usage:**
```bash
# Generate stock chart
chart-generator generate candlestick --template stock-performance --data stock-prices.json

# Data format
[
  {
    "date": "2024-01-01",
    "open": 150.25,
    "high": 152.50,
    "low": 148.75,
    "close": 151.00,
    "volume": 1000000
  }
]
```

#### 5. Portfolio Asset Allocation
**ID:** `portfolio-allocation`  
**Chart Type:** Treemap  
**Industry:** Investment  
**Best For:** Investment portfolio breakdown by asset class

**Usage:**
```bash
# Visualize portfolio allocation
chart-generator generate treemap --template portfolio-allocation --data portfolio.json

# Data format with hierarchical structure
[
  {"asset_class": "Equities", "value": 500000, "parent": "root"},
  {"asset": "US Stocks", "value": 300000, "parent": "Equities"},
  {"asset": "International", "value": 200000, "parent": "Equities"},
  {"asset_class": "Bonds", "value": 300000, "parent": "root"}
]
```

#### 6. Cash Flow Statement
**ID:** `cash-flow`  
**Chart Type:** Area Chart  
**Industry:** Corporate  
**Best For:** Operating, investing, and financing activities visualization

**Usage:**
```bash
# Generate cash flow visualization
chart-generator generate area --template cash-flow --data cashflow.json

# Data format
[
  {
    "month": "Jan",
    "operating": 50000,
    "investing": -20000,
    "financing": 10000
  }
]
```

### 🏥 Healthcare Templates

Healthcare templates optimize for medical and patient data visualization.

#### 7. Patient Metrics Dashboard
**ID:** `patient-metrics`  
**Chart Type:** Heatmap  
**Industry:** Hospital  
**Best For:** Patient volume by department and time of day

**Usage:**
```bash
# Create patient volume heatmap
chart-generator generate heatmap --template patient-metrics --data patient-volume.json

# Data format
[
  {
    "department": "Emergency",
    "hour": "08:00",
    "patient_count": 45
  },
  {
    "department": "Emergency",
    "hour": "14:00",
    "patient_count": 62
  }
]
```

#### 8. Treatment Outcome Analysis
**ID:** `treatment-outcomes`  
**Chart Type:** Scatter Plot  
**Industry:** Clinical  
**Best For:** Treatment effectiveness correlation analysis

**Usage:**
```bash
# Analyze treatment correlations
chart-generator generate scatter --template treatment-outcomes --data outcomes.json

# Data format
[
  {
    "treatment_duration_days": 30,
    "improvement_percentage": 75,
    "patient_age": 45
  }
]
```

### 💻 Technology Templates

Technology templates focus on software development and system metrics.

#### 9. System Performance Metrics
**ID:** `system-performance`  
**Chart Type:** Line Chart  
**Industry:** DevOps  
**Best For:** CPU, memory, and network utilization tracking

**Usage:**
```bash
# Monitor system metrics
chart-generator generate line --template system-performance --data metrics.json

# Data format
[
  {
    "timestamp": "2024-01-01T00:00:00Z",
    "cpu_percent": 45,
    "memory_percent": 62,
    "network_mbps": 120
  }
]
```

#### 10. Sprint Burndown Chart
**ID:** `sprint-burndown`  
**Chart Type:** Area Chart  
**Industry:** Software  
**Best For:** Agile sprint progress tracking

**Usage:**
```bash
# Track sprint progress
chart-generator generate area --template sprint-burndown --data sprint-data.json

# Data format
[
  {
    "day": 1,
    "remaining_points": 100,
    "ideal_remaining": 100
  },
  {
    "day": 2,
    "remaining_points": 85,
    "ideal_remaining": 90
  }
]
```

#### 11. Deployment Timeline
**ID:** `deployment-timeline`  
**Chart Type:** Gantt  
**Industry:** Software  
**Best For:** Release schedule and dependencies

**Usage:**
```bash
# Create deployment schedule
chart-generator generate gantt --template deployment-timeline --data releases.json

# Data format
[
  {
    "release": "v1.0",
    "component": "Backend API",
    "start": "2024-01-01",
    "end": "2024-01-15",
    "dependencies": []
  },
  {
    "release": "v1.0",
    "component": "Frontend",
    "start": "2024-01-10",
    "end": "2024-01-25",
    "dependencies": ["Backend API"]
  }
]
```

### 📈 Marketing Templates

Marketing templates optimize for campaign and customer analytics.

#### 12. Campaign Performance Matrix
**ID:** `campaign-performance`  
**Chart Type:** Heatmap  
**Industry:** Advertising  
**Best For:** Multi-channel campaign effectiveness visualization

**Usage:**
```bash
# Analyze campaign performance
chart-generator generate heatmap --template campaign-performance --data campaigns.json

# Data format
[
  {
    "channel": "Email",
    "campaign": "Summer Sale",
    "conversion_rate": 3.5,
    "roi": 250
  },
  {
    "channel": "Social Media",
    "campaign": "Summer Sale",
    "conversion_rate": 2.1,
    "roi": 180
  }
]
```

#### 13. Customer Journey Funnel
**ID:** `customer-journey`  
**Chart Type:** Bar Chart  
**Industry:** E-commerce  
**Best For:** Conversion funnel visualization

**Usage:**
```bash
# Visualize conversion funnel
chart-generator generate bar --template customer-journey --data funnel.json

# Data format
[
  {"stage": "Awareness", "users": 10000, "percentage": 100},
  {"stage": "Interest", "users": 6000, "percentage": 60},
  {"stage": "Consideration", "users": 3000, "percentage": 30},
  {"stage": "Purchase", "users": 500, "percentage": 5}
]
```

### 🎓 Education Templates

Education templates focus on academic performance and enrollment analytics.

#### 14. Student Performance Distribution
**ID:** `student-performance`  
**Chart Type:** Scatter Plot  
**Industry:** Academic  
**Best For:** Grade distribution and trend analysis

**Usage:**
```bash
# Analyze student grades
chart-generator generate scatter --template student-performance --data grades.json

# Data format
[
  {
    "student_id": "S001",
    "attendance_rate": 95,
    "final_grade": 88,
    "study_hours_week": 15
  }
]
```

#### 15. Course Enrollment Trends
**ID:** `course-enrollment`  
**Chart Type:** Area Chart  
**Industry:** University  
**Best For:** Enrollment patterns over semesters

**Usage:**
```bash
# Track enrollment trends
chart-generator generate area --template course-enrollment --data enrollment.json

# Data format
[
  {
    "semester": "Fall 2023",
    "undergraduate": 5000,
    "graduate": 1500,
    "doctoral": 300
  },
  {
    "semester": "Spring 2024",
    "undergraduate": 4800,
    "graduate": 1600,
    "doctoral": 320
  }
]
```

## Using Templates Effectively

### Template Selection Guide

| If you need to... | Use this template |
|-------------------|-------------------|
| Compare quarterly business metrics | `quarterly-sales` |
| Show trends over time | `revenue-trend` |
| Display proportional data | `market-share` |
| Visualize stock prices | `stock-performance` |
| Show hierarchical financial data | `portfolio-allocation` |
| Track cash flow categories | `cash-flow` |
| Display intensity across dimensions | `patient-metrics` or `campaign-performance` |
| Show correlations | `treatment-outcomes` or `student-performance` |
| Monitor system metrics | `system-performance` |
| Track project progress | `sprint-burndown` |
| Plan project timelines | `deployment-timeline` |
| Visualize conversion funnels | `customer-journey` |
| Show enrollment patterns | `course-enrollment` |

### Customizing Templates

Templates can be customized by overriding default settings:

```bash
# Use template with custom styling
chart-generator generate bar --template quarterly-sales --style vibrant --data sales.json

# Override template dimensions
chart-generator generate line --template revenue-trend --width 1200 --height 800 --data revenue.json

# Add custom title to template
chart-generator generate pie --template market-share --title "2024 Market Position" --data market.json
```

### API Template Usage

```javascript
// Using templates via API
const response = await fetch('http://localhost:19095/api/v1/charts/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    template_id: 'quarterly-sales',
    data: salesData,
    config_overrides: {
      title: 'Q4 2024 Sales Performance',
      colors: ['#0369a1', '#10b981', '#f59e0b'],
      show_legend: true
    }
  })
});
```

### Batch Processing with Templates

```bash
#!/bin/bash
# Generate multiple reports using templates

# Financial reports
for month in Jan Feb Mar; do
  chart-generator generate candlestick \
    --template stock-performance \
    --data "stock-${month}.json" \
    --output "reports/${month}-stocks.png"
done

# Department metrics
for dept in sales marketing engineering; do
  chart-generator generate bar \
    --template quarterly-sales \
    --data "${dept}-quarterly.json" \
    --title "${dept^} Department Performance" \
    --output "dashboards/${dept}.png"
done
```

## Template Best Practices

### 1. Data Preparation
- Ensure data matches the expected format for each template
- Validate data before generation to avoid errors
- Use consistent date formats (ISO 8601 recommended)

### 2. Performance Optimization
- Templates are pre-optimized for their specific use cases
- Avoid overriding performance-critical settings unless necessary
- Use appropriate data aggregation for large datasets

### 3. Visual Consistency
- Templates within the same category share visual elements
- Use consistent templates across related reports
- Apply company branding through custom styles

### 4. Export Considerations
- Financial templates default to high-resolution exports
- Healthcare templates ensure HIPAA-compliant styling
- Education templates optimize for print readability

## Creating Custom Templates

While the 15 built-in templates cover most use cases, you can create custom templates:

```bash
# Save current configuration as template
curl -X POST http://localhost:19095/api/v1/templates/create \
  -d '{
    "name": "Custom Sales Dashboard",
    "category": "business",
    "chart_type": "composite",
    "default_config": {
      "layout": "grid",
      "charts": [...],
      "styling": {...}
    }
  }'
```

## Template Troubleshooting

### Common Issues

1. **Data format mismatch**
   - Check that field names match template expectations
   - Verify data types (numbers vs strings)
   - Ensure required fields are present

2. **Styling conflicts**
   - Template styles take precedence over defaults
   - Use `config_overrides` to customize
   - Check style compatibility with chart type

3. **Performance issues**
   - Large datasets may need aggregation
   - Consider using data transformation endpoints
   - Enable caching for repeated generations

For more details, see the [API Documentation](api.md) or run `chart-generator help`.