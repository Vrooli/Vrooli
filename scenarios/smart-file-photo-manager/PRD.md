# Smart File Photo Manager - Product Requirements Document

## Executive Summary
The Smart File Photo Manager is an AI-powered file organization system with specialized photo management capabilities. It provides intelligent categorization, semantic search, duplicate detection, and visual organization tools for managing large photo and document collections. Primary users are individuals and small businesses needing automated file organization. Business value: $15K-25K as a SaaS offering or enterprise deployment. Priority: High (addresses universal pain point).

## Requirements Checklist

### P0 Requirements (Must Have - Core Functionality)
- [x] **Health Check**: API responds with health status (verified: working at /health on port 16025)
- [x] **Lifecycle Management**: setup/develop/test/stop commands work via Makefile (verified: all working)
- [x] **Database Integration**: Files can be stored and retrieved from PostgreSQL (verified: working with simplified schema)
- [x] **API Endpoints**: Core REST API for file operations (verified: /api/files working)
- [ ] **File Upload**: Support file upload with metadata extraction (PARTIAL: endpoint exists but schema issues block)
- [x] **Basic Search**: Text-based search across file metadata (verified: GET and POST /api/search working)
- [ ] **Folder Organization**: Create and manage folder hierarchies (NOT IMPLEMENTED: returns 501)

### P1 Requirements (Should Have - Enhanced Features)
- [ ] **AI Processing**: Integration with Ollama for image analysis
- [ ] **Duplicate Detection**: Find similar/duplicate files
- [ ] **Semantic Search**: Vector-based similarity search using Qdrant
- [ ] **UI Gallery**: Visual photo gallery with thumbnails

### P2 Requirements (Nice to Have - Advanced)
- [ ] **Auto-Tagging**: AI-generated tags based on content
- [ ] **Smart Categorization**: Automatic folder suggestions
- [ ] **Batch Operations**: Process multiple files simultaneously

## Core Features

### 1. Photo Gallery Interface
- **Grid Layout**: Clean, responsive grid layout optimized for image browsing
- **Image Previews**: Automatic thumbnail generation and preview display
- **Folder Organization**: Intuitive folder tree structure with drag-and-drop organization
- **View Modes**: Toggle between grid and list views for different use cases
- **Modal Preview**: Full-screen modal for detailed image viewing with metadata

### 2. AI-Powered Organization
- **Semantic Search**: Natural language search using vector embeddings
- **Auto-Tagging**: AI-generated tags based on image content and metadata
- **Smart Categorization**: Automatic folder suggestions based on content analysis
- **Duplicate Detection**: Advanced duplicate detection using image similarity

### 3. File Management
- **Drag & Drop Upload**: Intuitive file upload with progress indicators
- **Batch Operations**: Select and process multiple files simultaneously
- **File Metadata**: Display comprehensive file information (size, date, type, etc.)
- **Format Support**: Support for images, documents, videos, and audio files

### 4. User Experience
- **Modern UI**: Clean, professional interface with smooth animations
- **Responsive Design**: Works seamlessly across desktop and mobile devices
- **Real-time Updates**: Live updates as files are processed and organized
- **Visual Feedback**: Clear indicators for AI processing and upload progress

## UI Design Specifications

### Visual Design
- **Color Scheme**: Modern blue-purple gradient primary colors (#6366f1, #818cf8)
- **Typography**: Inter font family for clean readability
- **Layout**: Sidebar navigation with main content area
- **Spacing**: Generous whitespace for clean, uncluttered appearance

### Component Structure
1. **Sidebar Navigation**
   - Logo and branding
   - Navigation menu (Organize, Search, Duplicates, Tags, Insights)
   - Storage usage indicator

2. **Header Bar**
   - Semantic search input
   - Upload button
   - AI organize button

3. **Main Content Area**
   - View-specific panels
   - File grid/list display
   - Upload drop zone

4. **Modal Components**
   - File preview modal
   - AI processing indicators

### Photo Management Focus
- **Gallery Grid**: Optimized 200px minimum column width for photo thumbnails
- **Image Previews**: 150px height preview cards with proper aspect ratio handling
- **Folder Views**: Visual folder organization with image count indicators
- **Metadata Display**: EXIF data and AI-generated descriptions for photos

### Interactive Elements
- **Hover Effects**: Subtle transform and shadow effects on cards
- **Loading States**: Animated pulse rings for AI processing
- **Drag & Drop**: Visual feedback for file upload areas
- **View Toggles**: Grid/list view switching with smooth transitions

## Technical Specifications

### Architecture
- **API**: Go (Gin framework) - Port 16035 ✅
- **UI**: Node.js/Express with HTML/CSS/JS - Port 38743
- **Database**: PostgreSQL (simplified schema without pgvector) ✅
- **Cache**: Redis (configured but not fully integrated) ⚠️
- **File Storage**: MinIO (configured but not tested) ⚠️
- **AI**: Ollama (configured but not integrated) ⚠️
- **Vector DB**: Qdrant (configured but collections not initialized) ⚠️

### Dependencies Status
- PostgreSQL: ✅ Running and integrated
- Redis: ✅ Running with basic integration
- MinIO: ⚠️ Running but not tested
- Qdrant: ⚠️ Running but not initialized
- Ollama: ⚠️ Configured but not tested
- Unstructured-IO: ❌ Not integrated

### API Endpoints (Current State)
- GET /health - ✅ Health check working
- GET /api/files - ✅ List files working
- GET /api/files/:id - ✅ Get file details
- POST /api/files - ⚠️ Upload file (not tested)
- PUT /api/files/:id - ⚠️ Update file (not tested)
- DELETE /api/files/:id - ⚠️ Delete file (not tested)
- GET /api/folders - ✅ List folders working
- POST /api/search - ⚠️ Search files (expects POST, test uses GET)
- GET /api/search - ✅ Search endpoint available

## Success Metrics

### Completion Status
- P0 Requirements: 71% (5/7 completed, 1 partial)
- P1 Requirements: 0% (0/4 completed)
- P2 Requirements: 0% (0/3 completed)
- Overall Progress: 43% (6/14 features working or partial)

### Quality Metrics
- Health Check Response Time: <10ms ✅
- API Response Time: <100ms for basic queries ✅
- Test Pass Rate: TBD (tests need updating)
- Database Schema: Simplified (no vector support)

### Performance Targets
- Page Load: <2 seconds (not tested)
- File Upload: >10MB/s (not tested)
- Search Response: <500ms (not working)
- Concurrent Users: 100+ (not tested)

## Future Enhancements

1. **Advanced Photo Features**
   - Face recognition and grouping
   - Location-based organization
   - Timeline view for photos
   - Photo editing capabilities

2. **Collaboration**
   - Shared photo albums
   - Comment and annotation system
   - User permissions and roles

3. **Integration**
   - Cloud storage providers (Google Drive, Dropbox, etc.)
   - Social media import
   - External photo services integration

## Implementation History

### 2025-09-24 Progress (17:00-17:30)
- Fixed Go build error (removed unused imports)
- Fixed environment variable handling (added fallbacks for QDRANT_URL, MINIO_URL, OLLAMA_URL, REDIS_URL)
- Created simplified database schema without pgvector extension
- Fixed missing database columns (current_name, file_hash, updated_at, etc.)
- Verified API health check and basic endpoints working
- Identified blocking issues with vector search and AI integration

### 2025-09-30 Progress (01:00-01:10)
- Confirmed API running on dynamic ports (16025-16035 range)
- Verified health check endpoint working (<10ms response time)
- Verified files list endpoint working (returns empty list correctly)
- Fixed search endpoint - both GET and POST methods working
- Attempted to fix file upload - database schema inconsistency blocks progress
- Added dynamic schema detection to handle multiple database schemas
- Updated PRD with proper requirements tracking format
- Progress: 29% → 43% (health, lifecycle, database, API endpoints, search working)

## Revenue Justification
- **Market Size**: 500M+ users struggle with photo/file organization
- **Pricing Model**: $10-20/month SaaS or $15K enterprise license
- **Competition**: Google Photos ($10/mo), Dropbox ($12/mo), custom solutions ($50K+)
- **Unique Value**: Local-first, AI-powered, privacy-focused, customizable
- **Revenue Potential**: 1000 users × $15/mo = $180K/year or 10 enterprise × $15K = $150K