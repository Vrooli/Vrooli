# SEO Optimizer

## Overview
A comprehensive SEO analysis and optimization platform that helps improve search engine rankings through automated audits, content optimization, keyword research, and competitor analysis.

## Purpose
This scenario provides businesses and content creators with professional-grade SEO tools to:
- Conduct thorough website SEO audits
- Optimize content for target keywords
- Research high-value keywords
- Analyze competitor SEO strategies
- Track ranking improvements over time

## Key Features
- **SEO Audit**: Comprehensive analysis of on-page and technical SEO factors
- **Content Optimizer**: Real-time content optimization suggestions
- **Keyword Research**: Discover high-value keywords and search opportunities
- **Competitor Analysis**: Compare SEO performance with competitors
- **Ranking Tracker**: Monitor search engine rankings over time

## Dependencies
This scenario relies on:
- **Ollama**: For AI-powered SEO analysis and content generation
- **Browserless**: For web scraping and screenshot capture
- **PostgreSQL**: For storing audit history and keyword data
- **Redis**: For caching analysis results
- **Qdrant**: For semantic content analysis (optional)
- **N8n**: For workflow automation

## Shared Workflows Used
- `ollama.json`: AI text generation for SEO analysis
- `embedding-generator.json`: Content embeddings for semantic search
- `structured-data-extractor.json`: Extract structured SEO data

## UX Design
Professional, clean interface optimized for business use:
- Modern, minimalist design with data visualization
- Dashboard-style layout with clear metrics
- Professional color scheme (blues, grays)
- Clear navigation between SEO tools
- Real-time analysis feedback

## Use Cases
1. **Digital Marketing Agencies**: Provide SEO services to clients
2. **Content Teams**: Optimize blog posts and web content
3. **E-commerce Sites**: Improve product page rankings
4. **Small Businesses**: DIY SEO improvements
5. **SEO Consultants**: Professional SEO auditing tool

## API Endpoints
- `POST /api/seo-audit`: Run comprehensive SEO audit
- `POST /api/content-optimize`: Optimize content for SEO
- `POST /api/keyword-research`: Research keywords
- `POST /api/competitor-analysis`: Analyze competitors

## Future Enhancements
- Backlink analysis integration
- Page speed optimization recommendations
- Mobile SEO analysis
- Voice search optimization
- Local SEO features
- Automated reporting system