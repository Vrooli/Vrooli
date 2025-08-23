# Research Assistant Scenario - Implementation Plan

## Overview
Transform the research-assistant scenario into a powerful, DeepSearch-quality research tool with dashboard, chatbot, scheduling, and PDF export capabilities.

## Core Requirements
- **Dashboard**: View and manage research reports
- **Chatbot**: Conversational interface to discuss reports and request new research
- **Scheduling**: Automated recurring reports (e.g., daily AI progress updates)
- **PDF Export**: Professional report generation
- **Deep Research**: Multi-source verification with configurable depth
- **Local Storage**: Reports saved in PostgreSQL with embeddings in Qdrant

## Resource Architecture

### Core Intelligence
- **Ollama (11434)**: qwen2.5:32b for analysis, nomic-embed-text for embeddings
- **SearXNG (9200)**: Privacy-respecting meta-search across 70+ engines

### Orchestration & UI
- **n8n (5678)**: Research pipeline orchestration, scheduling, RAG workflow
- **Windmill (5681)**: Dashboard, chat interface, report viewer

### Processing & Storage
- **Unstructured-IO (11450)**: Source processing, PDF generation
- **Browserless (4110)**: JavaScript-heavy site extraction
- **PostgreSQL (5433)**: Report metadata, schedules, chat history
- **Qdrant (6333)**: Vector embeddings for semantic search
- **MinIO (9000)**: PDF storage, source cache

## Key Workflows

### 1. Research Pipeline (n8n)
1. Receive request with topic, depth (quick/standard/deep), length (1-10 pages)
2. Initial SearXNG search → 20-50 results
3. Query refinement → focused searches
4. Browserless extraction of top sources
5. Unstructured-IO source processing
6. Ollama analysis: themes, contradictions, consensus
7. Generate markdown report with citations
8. Create embeddings, store in Qdrant
9. Convert to PDF, store in MinIO

### 2. Chat Interface
- Windmill UI → n8n RAG workflow
- Retrieve relevant report chunks from Qdrant
- Ollama generates contextual responses
- Can trigger new research from chat
- Maintains conversation history

### 3. Scheduling System
- Cron triggers for scheduled reports
- Pre-configured templates
- Customizable parameters per schedule
- Automatic PDF generation and storage

### 4. Dashboard Features
- Report gallery with search/filter
- Quick action buttons
- Schedule overview
- Integrated chat panel
- Settings management

## Quality Standards
- Minimum 5 sources per claim
- Contradiction detection and highlighting
- Confidence scoring for insights
- Source quality ranking

## File Structure
```
research-assistant/
├── .vrooli/service.json
├── initialization/
│   ├── automation/
│   │   ├── n8n/
│   │   │   ├── research-orchestrator.json
│   │   │   ├── scheduled-reports.json
│   │   │   └── chat-rag-workflow.json
│   │   └── windmill/
│   │       ├── dashboard-app.json
│   │       ├── chat-interface.json
│   │       ├── report-viewer.json
│   │       └── schedule-manager.json
│   ├── agents/
│   │   └── browserless/
│   │       └── extraction-config.json
│   ├── ai/
│   │   ├── ollama/
│   │   │   └── models.json
│   │   └── unstructured-io/
│   │       └── pdf-templates.json
│   ├── configuration/
│   │   ├── research-config.json
│   │   ├── report-templates.json
│   │   └── schedule-presets.json
│   ├── search/
│   │   └── searxng/
│   │       └── engines-config.json
│   └── storage/
│       ├── postgres/
│       │   ├── schema.sql
│       │   └── seed.sql
│       ├── qdrant/
│       │   └── collections.json
│       └── minio/
│           └── buckets.json
├── deployment/
│   ├── startup.sh
│   └── monitor.sh
└── test.sh
```

## Implementation Steps
1. Update service.json with all required resources
2. Create database schema for reports, schedules, chat history
3. Build n8n workflows for research pipeline
4. Create Windmill UI applications
5. Configure resource integrations
6. Set up deployment and validation scripts
7. Create comprehensive tests

## Value Proposition
Enterprise-grade research platform competing with Perplexity Pro, Elicit, Consensus while maintaining complete data privacy. Estimated value: $30-50K per deployment.