# Smart File Photo Manager - Product Requirements Document

## Executive Summary
The Smart File Photo Manager is an AI-powered file organization system with specialized photo management capabilities. It provides intelligent categorization, semantic search, duplicate detection, and visual organization tools for managing large photo and document collections. Primary users are individuals and small businesses needing automated file organization. Business value: $15K-25K as a SaaS offering or enterprise deployment. Priority: High (addresses universal pain point).

## Requirements Checklist

### P0 Requirements (Must Have - Core Functionality)
- [x] **Health Check**: API responds with health status (tested: working at /health)
- [x] **Lifecycle Management**: setup/develop/test/stop commands work via Makefile (tested: all working)
- [x] **Database Integration**: Files can be stored and retrieved from PostgreSQL (tested: working with simplified schema)
- [x] **API Endpoints**: Core REST API for file operations (tested: /api/files working)
- [ ] **File Upload**: Support file upload with metadata extraction
- [ ] **Basic Search**: Text-based search across file metadata  
- [ ] **Folder Organization**: Create and manage folder hierarchies

### P1 Requirements (Should Have - Enhanced Features)
- [ ] **AI Processing**: Integration with Ollama for image analysis
- [ ] **Duplicate Detection**: Find similar/duplicate files
- [ ] **Semantic Search**: Vector-based similarity search using Qdrant
- [ ] **UI Gallery**: Visual photo gallery with thumbnails

### P2 Requirements (Nice to Have - Advanced)
- [ ] **Auto-Tagging**: AI-generated tags based on content
- [ ] **Smart Categorization**: Automatic folder suggestions
- [ ] **Batch Operations**: Process multiple files simultaneously

## Technical Specifications

### Architecture
- **API**: Go (Gin framework) - Port 16029
- **UI**: Node.js/Express with HTML/CSS/JS - Port 38743  
- **Database**: PostgreSQL (simplified schema without pgvector)
- **Cache**: Redis (configured but not fully integrated)
- **File Storage**: MinIO (configured but not tested)
- **AI**: Ollama (configured but not integrated)
- **Vector DB**: Qdrant (configured but collections not initialized)

### Dependencies
- PostgreSQL: ✅ Running and integrated
- Redis: ✅ Running with basic integration
- MinIO: ⚠️ Running but not tested
- Qdrant: ⚠️ Running but not initialized
- Ollama: ⚠️ Configured but not tested
- Unstructured-IO: ❌ Not integrated

### API Endpoints (Implemented)
- GET /health - ✅ Health check
- GET /api/files - ✅ List files
- GET /api/files/:id - ✅ Get file details
- POST /api/files - ⚠️ Upload file (not tested)
- PUT /api/files/:id - ⚠️ Update file (not tested)
- DELETE /api/files/:id - ⚠️ Delete file (not tested)
- GET /api/folders - ✅ List folders
- POST /api/search - ⚠️ Search files (expects POST, not fully working)

## Success Metrics

### Completion Status
- P0 Requirements: 57% (4/7 completed)
- P1 Requirements: 0% (0/4 completed)  
- P2 Requirements: 0% (0/3 completed)
- Overall Progress: 29% (4/14 features)

### Quality Metrics
- Health Check Response Time: <50ms ✅
- API Response Time: <100ms for basic queries ✅
- Test Pass Rate: 50% (3/6 tests passing)
- Database Schema: Simplified (no vector support)

### Performance Targets
- Page Load: <2 seconds (not tested)
- File Upload: >10MB/s (not tested)
- Search Response: <500ms (not working)
- Concurrent Users: 100+ (not tested)

## Implementation History

### 2025-09-24 Progress (17:00-17:30)
- Fixed Go build error (removed unused imports)
- Fixed environment variable handling (added fallbacks for QDRANT_URL, MINIO_URL, OLLAMA_URL, REDIS_URL) 
- Created simplified database schema without pgvector extension
- Fixed missing database columns (current_name, file_hash, updated_at, etc.)
- Verified API health check and basic endpoints working
- Identified blocking issues with vector search and AI integration

### Current State
- **Working**: Lifecycle, health check, basic API, database queries
- **Degraded**: Search functionality (POST only), no vector support
- **Not Working**: File upload, AI processing, duplicate detection, UI functionality
- **Blockers**: pgvector extension missing, Qdrant collections not initialized, file processing not tested

## Next Steps for Future Improvements

### Immediate (P0 Fixes)
1. Test and fix file upload endpoint
2. Implement basic text search without vectors
3. Test folder management operations
4. Fix UI to display files correctly

### Short-term (P1 Implementation)  
1. Initialize Qdrant collections for vector search
2. Integrate Ollama for image analysis
3. Implement duplicate detection algorithm
4. Build photo gallery UI

### Long-term (P2 Features)
1. Add pgvector support when available
2. Implement auto-tagging system
3. Build smart categorization engine
4. Add batch processing capabilities

## Revenue Justification
- **Market Size**: 500M+ users struggle with photo/file organization
- **Pricing Model**: $10-20/month SaaS or $15K enterprise license
- **Competition**: Google Photos ($10/mo), Dropbox ($12/mo), custom solutions ($50K+)
- **Unique Value**: Local-first, AI-powered, privacy-focused, customizable
- **Revenue Potential**: 1000 users × $15/mo = $180K/year or 10 enterprise × $15K = $150K