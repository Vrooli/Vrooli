# Smart File Photo Manager - Known Issues

## Critical Issues

### 1. Database Schema Mismatch (P0)
**Status**: Blocking file upload functionality
**Severity**: High
**Impact**: File upload endpoint returns database constraint violations

**Problem**:
The database schema deployed does not match either the official `schema.sql` or `schema-simple.sql` files. The deployed schema has columns like `filename` and `size` that are NOT NULL, while the code expects `original_name` and `size_bytes`.

**Evidence**:
```
Error: "pq: null value in column \"filename\" of relation \"files\" violates not-null constraint"
Error: "pq: null value in column \"size\" of relation \"files\" violates not-null constraint"
```

**Root Cause**:
1. service.json:46-51 references `initialization/storage/postgres/schema.sql` (full schema with vector extensions)
2. The actual deployed schema appears to be from a different version or source
3. Dynamic schema detection in `uploadFile()` tries to handle this but can't resolve all mismatches

**Impact**:
- File upload endpoint (POST /api/files) is non-functional
- Cannot test AI processing pipeline
- Cannot demonstrate core file management features

**Recommended Fix**:
1. Drop and recreate the `files` table using `schema-simple.sql` (which doesn't require pgvector extension)
2. Update service.json to reference `schema-simple.sql` instead of `schema.sql`
3. Run `vrooli scenario smart-file-photo-manager setup --force` to reinitialize

**Workaround**:
File upload is disabled until schema is fixed. Other endpoints (health, search, folders) work correctly.

---

## P1 Issues

### 2. Qdrant Collection Initialization Failed
**Status**: Non-blocking (semantic search unavailable)
**Severity**: Medium
**Impact**: Semantic search and AI-powered features unavailable

**Problem**:
During scenario setup, Qdrant collection initialization fails:
```
ERROR: ❌ Failed to add: (collection)
WARNING: Failed to populate qdrant (resource may not be running)
```

**Impact**:
- Semantic search returns basic text search results only
- Vector embeddings cannot be stored
- AI similarity features unavailable

**Recommended Fix**:
1. Ensure Qdrant resource is running before scenario setup
2. Check `initialization/storage/qdrant/collections.json` for valid collection definitions
3. Verify Qdrant connection string in environment variables

---

## P2 Issues

### 3. Vector Extension Missing
**Status**: Documented limitation
**Severity**: Low
**Impact**: Advanced AI features require schema migration

**Problem**:
The current deployment uses `schema-simple.sql` which intentionally omits the pgvector extension to simplify setup. This means vector embedding columns are not available.

**Impact**:
- Cannot store content_embedding, visual_embedding columns
- Semantic search relies on text matching only
- Image similarity detection unavailable

**Solution**:
This is by design for the simplified deployment. For production, migrate to full `schema.sql` with pgvector extension installed on PostgreSQL.

---

## Testing & Validation

### 4. Test Suite Incomplete
**Status**: In progress
**Severity**: Medium

**Missing Coverage**:
- File upload endpoint tests (blocked by schema issue)
- AI processing pipeline tests
- Integration tests with Ollama/Qdrant
- UI automation tests

**Next Steps**:
1. Fix schema issue (#1)
2. Create test script for file upload workflow
3. Add BATS tests for CLI commands
4. Implement phased testing structure per docs/scenarios/PHASED_TESTING_ARCHITECTURE.md

---

## Resolved Issues

### ✅ Folder Endpoints Implemented (2025-10-03)
**Status**: Fixed
**Original Issue**: Folder management endpoints returned 501 Not Implemented

**Solution**: Implemented all folder CRUD operations:
- GET /api/folders - List all folders ✅
- POST /api/folders - Create folder ✅
- PUT /api/folders/:path - Update folder ✅
- DELETE /api/folders/:path - Delete folder (with safety check) ✅

**Validation**:
```bash
curl http://localhost:16025/api/folders
# Returns: {"folders":[...],"total":1}

curl -X POST http://localhost:16025/api/folders -H "Content-Type: application/json" \
  -d '{"path":"/Photos","parent_path":"/","name":"Photos","description":"Photo collection"}'
# Returns: {"folder_id":"...", "message":"Folder created successfully"}
```

---

## Recommendations

1. **Immediate**: Fix database schema mismatch to unblock file upload
2. **Short-term**: Initialize Qdrant collections for semantic search
3. **Medium-term**: Implement comprehensive test suite
4. **Long-term**: Migrate to full schema.sql with vector support for production deployment
