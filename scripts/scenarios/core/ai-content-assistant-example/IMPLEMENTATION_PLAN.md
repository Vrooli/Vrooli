# AI Content Assistant Implementation Plan

## Overview
Transform the generic AI assistant into a campaign-based content management system with AI-powered content generation, document context management, and multi-format content creation.

## Target Functionality
- **Campaign Management**: Tab-based interface for organizing content projects
- **Document Context**: Upload and embed documents for campaign-specific context
- **Semantic Search**: Find relevant document excerpts using vector similarity
- **AI Content Generation**: Generate blog posts, social media content, images using campaign context
- **Content History**: Track and manage all generated content per campaign

## Required Resources

### Core Infrastructure
- **PostgreSQL (5433)**: Campaign metadata, document tracking, content history
- **Qdrant (6333)**: Document embeddings for semantic search
- **MinIO (9000)**: File storage for documents and generated assets
- **Ollama (11434)**: Text generation with context-aware prompts
- **n8n (5678)**: Workflow orchestration for document processing and content generation
- **Windmill (5681)**: Main dashboard UI with campaign management

### Content Generation
- **ComfyUI (8188)**: Image generation for blog posts and social media
- **Unstructured-IO (11450)**: Document text extraction from PDFs, DOCX, etc.

## Database Schema Design

### Core Tables
```sql
-- Campaign management
campaigns (id, name, description, settings, created_at)

-- Document storage and metadata  
campaign_documents (id, campaign_id, filename, file_path, content_type, processed_text, embedding_id, upload_date)

-- Content generation history
generated_content (id, campaign_id, content_type, prompt, generated_content, used_documents, created_at)

-- Enhanced user sessions with campaign context
user_sessions (id, session_id, current_campaign_id, preferences, context_data)
```

## Key Workflows

### 1. Campaign Management
- **Create Campaign** → PostgreSQL storage → New tab in UI
- **Switch Campaign** → Load campaign context → Update document list
- **Campaign Settings** → Style guides, content preferences

### 2. Document Processing Pipeline
```
Upload Document → MinIO Storage → Unstructured-IO Text Extraction → 
Ollama Embeddings → Qdrant Vector Storage → PostgreSQL Metadata
```

### 3. Content Generation Flow
```
Select Campaign → Qdrant Context Retrieval → User Prompt + Context → 
Ollama Generation → Optional ComfyUI Images → PostgreSQL History
```

### 4. Search & Context Management
- **Semantic Search**: Query campaign documents using Qdrant similarity
- **Context Preview**: Show relevant excerpts before generation
- **Document Management**: Include/exclude specific documents from context

## UI Components (Windmill Dashboard)

### Main Layout
- **Campaign Tabs**: Dynamic tabs for each campaign
- **Document Manager**: Upload area, document list with search
- **Context Viewer**: Relevant document excerpts for current prompt
- **Content Generator**: Text input with content type selection
- **Generation History**: Previous content with regeneration options
- **Settings Panel**: Campaign-specific preferences

### Content Types Supported
- Blog Posts (structured with headings, SEO optimization)
- Social Media Posts (Twitter, LinkedIn, Instagram formats)
- Marketing Copy (emails, ads, landing pages)
- Images (blog headers, social media graphics)

## n8n Workflows

### 1. campaign-management.json
- Campaign CRUD operations
- Campaign switching logic
- Settings management

### 2. document-processing.json
- File upload handling
- Text extraction via Unstructured-IO
- Embedding generation and Qdrant storage
- Metadata persistence

### 3. content-generation.json
- Context retrieval from Qdrant
- Prompt construction with campaign context
- Ollama text generation
- Optional ComfyUI image generation
- Result storage and formatting

### 4. search-retrieval.json
- Semantic search within campaign documents
- Context ranking and filtering
- Document excerpt extraction

## Business Value

### Target Market
- Content creators and marketing agencies
- Small to medium businesses with content needs
- Freelance writers and consultants
- Marketing teams managing multiple campaigns

### Value Proposition
- **Efficiency**: AI-powered content generation with relevant context
- **Organization**: Campaign-based project management
- **Consistency**: Maintained brand voice through document context
- **Scalability**: Handle multiple campaigns simultaneously
- **Intelligence**: Semantic search finds relevant context automatically

### Revenue Potential
- **Initial Setup**: $25,000 (complete content management system)
- **Monthly SaaS**: $50-200/month per team
- **Enterprise**: $500-2000/month for large agencies
- **Total Estimated Value**: $45,000-75,000

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] Update service.json with all required resources
- [ ] Create enhanced database schema
- [ ] Set up basic campaign management workflow

### Phase 2: Document Management
- [ ] Implement document upload and processing pipeline
- [ ] Set up Qdrant collections and embedding generation
- [ ] Create document search and context retrieval

### Phase 3: Content Generation
- [ ] Build context-aware content generation workflow
- [ ] Implement multiple content type templates
- [ ] Add ComfyUI image generation integration

### Phase 4: UI Development
- [ ] Create campaign tab interface in Windmill
- [ ] Build document management components
- [ ] Implement content generation interface with history

### Phase 5: Integration & Testing
- [ ] End-to-end workflow testing
- [ ] Performance optimization
- [ ] Business value validation

## Success Metrics
- Campaign creation and management functionality
- Document upload → embedding → search pipeline working
- Context-aware content generation producing relevant results
- Multi-format content generation (text + images)
- Professional UI supporting multiple campaigns simultaneously

## Technical Considerations
- **Vector Search**: Optimize Qdrant for sub-second search across campaign documents
- **Context Management**: Intelligent document chunking and relevance scoring
- **Content Templates**: Structured templates for different content types
- **Asset Management**: Organized storage of generated content and images
- **Performance**: Handle multiple concurrent content generation requests