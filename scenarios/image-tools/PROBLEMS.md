# Image Tools Problems & Solutions

## Problems Discovered

### 1. MinIO Storage Integration
**Problem**: The API falls back to local filesystem when MinIO is not running, but URLs still point to MinIO endpoints.
**Impact**: UI cannot display processed images when MinIO is unavailable.
**Solution**: Added proxy endpoint `/api/v1/image/proxy` to fetch images from storage backend.
**Status**: ✅ Resolved

### 2. UI Live Preview Not Working
**Problem**: Processed images were showing placeholders instead of actual results.
**Root Cause**: 
- MinIO URLs (http://localhost:9100/...) are not accessible directly from browser due to CORS
- File URLs (file://...) cannot be accessed from web context
**Solution**: 
- Added proxy endpoint to serve images through API
- Implemented fallback display with visual indicators
- Added dynamic API port detection
**Status**: ✅ Resolved

### 3. Dynamic Port Allocation
**Problem**: API port changes on each restart due to Vrooli lifecycle management.
**Impact**: UI loses connection to API after scenario restart.
**Solution**: Implemented dynamic port detection in UI JavaScript with fallback to common ports.
**Status**: ✅ Resolved

### 4. JPEG Decoding Errors
**Problem**: Some JPEG images fail with "invalid JPEG format: missing 0xff00 sequence".
**Root Cause**: Corrupted or non-standard JPEG format.
**Workaround**: Use PNG format for testing, validate JPEG files before processing.
**Status**: ⚠️ Known limitation

### 5. JQ Parsing Errors in Lifecycle
**Problem**: JQ errors appear during scenario startup.
**Root Cause**: Service.json structure doesn't match expected format for resource dependencies.
**Impact**: Cosmetic only - scenario still works correctly.
**Solution**: Migrated resources from array format to new object format with enabled/required/description fields.
**Status**: ✅ Resolved (2025-10-03)

### 6. Health Endpoint Compliance
**Problem**: UI and API health endpoints missing required schema fields.
**Root Cause**: Health endpoints implemented before schema requirements were defined.
**Impact**: Scenario status command shows warnings about invalid health responses.
**Solution**:
- Updated UI health endpoint to include `readiness` and `api_connectivity` fields with proper error handling
- Verified API health endpoint includes all required fields (`status`, `service`, `timestamp`, `readiness`)
**Status**: ✅ Resolved (2025-10-03)

## Recent Improvements (2025-10-03)

### 1. Preset Profiles System
**Feature**: Implemented reusable image processing profiles
**Details**:
- 5 built-in presets: web-optimized, email-safe, aggressive, high-quality, social-media
- New endpoints: GET /api/v1/presets, GET /api/v1/presets/:name, POST /api/v1/image/preset/:name
- Each preset defines operations chain (resize → compress → metadata strip)
**Benefit**: Users can apply common optimization patterns with a single API call

### 2. Service.json Migration
**Change**: Updated resources format from arrays to detailed object structure
**Before**: `"resources": { "required": ["minio"], "optional": ["redis", "ollama"] }`
**After**: Each resource has `type`, `enabled`, `required`, `description` fields
**Benefit**: Better compatibility with v2.0 lifecycle system, eliminates JQ parsing errors

## Recommendations for Future Improvements

1. **Implement proper CORS headers** for MinIO to allow direct browser access
2. **Add image validation** before processing to catch corrupt files early
3. **Implement caching layer** with Redis for frequently accessed images
4. **Add batch upload UI** with progress indicators
5. **Implement WebSocket** for real-time processing updates
6. **Add image history** tracking for undo/redo functionality
7. **Enhance preset system** with custom preset creation/storage via API
8. **Add AVIF format support** for next-gen image compression

## Performance Notes

- Current response time: ~10ms (target: <500ms) ✅
- Memory usage: ~12MB (target: <2GB) ✅
- Throughput: Not fully tested (target: 50 images/minute)
- Plugin loading: <100ms for all formats ✅

## Security Considerations

1. **File uploads**: Currently no size validation beyond 100MB limit
2. **Storage**: MinIO credentials are hardcoded defaults
3. **CORS**: Currently permissive for development
4. **Rate limiting**: Not implemented

## Dependencies Status

- ✅ Go 1.21+ (working)
- ✅ Fiber v2 framework (working)
- ✅ Plugin architecture (working)
- ⚠️ MinIO (optional, falls back to filesystem)
- ⚠️ Redis (optional, not yet integrated)
- ⚠️ Ollama (optional, for future AI features)