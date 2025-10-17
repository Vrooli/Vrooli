# ROI Fit Analysis

## Overview
ROI Fit Analysis is a sophisticated financial decision-making tool that performs deep web research to determine which business ideas, investments, or projects have the best return on investment based on your available skills, resources, and market conditions.

## Purpose
This scenario empowers entrepreneurs, investors, and business strategists to make data-driven decisions by:
- Analyzing market trends and competitive landscapes
- Evaluating resource requirements against available assets
- Calculating potential returns based on real market data
- Providing risk assessments and mitigation strategies

## Key Features
- **Market Intelligence**: Scrapes and analyzes current market data, competitor pricing, and industry trends
- **Skills Assessment**: Matches opportunities to your expertise and capabilities
- **Resource Optimization**: Identifies the most efficient use of capital, time, and human resources
- **Risk Analysis**: Quantifies potential risks and provides probability-weighted returns
- **Recommendation Engine**: Ranks opportunities by ROI potential with detailed justifications

## Dependencies
- **Shared Workflows**: 
  - `ollama.json` - For AI-powered analysis and insights
  - `embedding-generator.json` - For semantic search of market data
  - `structured-data-extractor.json` - For parsing financial reports and market data
- **Resources**: ollama, n8n, redis, qdrant, browserless

## UX Design
Professional, data-driven interface with a modern business aesthetic:
- Clean, minimalist design with dark mode for extended analysis sessions
- Interactive dashboards with real-time data visualization
- Professional color scheme (deep blues, grays, accent greens for positive ROI)
- Clear data tables, charts, and executive summaries
- Export functionality for reports and presentations

## Use Cases
- Startup founders evaluating multiple business ideas
- Investors comparing investment opportunities
- Corporate strategists assessing new market entries
- Consultants providing data-backed recommendations
- Small businesses deciding on expansion strategies

## Integration Points
- Can be enhanced by `product-manager-agent` for strategic planning
- Works with `competitor-change-monitor` for ongoing market intelligence
- Complements `mind-maps` for visual strategy planning