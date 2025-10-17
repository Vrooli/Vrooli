# FFmpeg Resource - Known Issues and Solutions

## Current Issues

### 1. Port Conflicts (FIXED)
**Issue**: Web interface defaulted to port 8080, conflicting with OpenTripPlanner
**Solution**: Changed default web port to 8098, API port remains 8097
**Status**: ✅ Fixed

### 2. Missing Input Validation in CLI (FIXED)
**Issue**: Some CLI commands don't validate input files exist before processing
**Solution**: Added file existence checks to media-info, transcode, and extract commands
**Status**: ✅ Fixed

### 3. Error Handling in API Server (FIXED)
**Issue**: Some API endpoints return generic 500 errors instead of specific error codes
**Solution**: Added detailed error responses with specific codes and troubleshooting hints
**Status**: ✅ Fixed

### 4. Hardware Acceleration Detection (FIXED)
**Issue**: GPU detection sometimes fails on certain NVIDIA driver versions
**Solution**: Enhanced detection now verifies NVENC encoder is available in FFmpeg, provides clear warnings and hints when GPU detected but encoder unavailable
**Status**: ✅ Fixed

### 5. Large File Processing Timeouts (FIXED)
**Issue**: Processing files >1GB may timeout with default 3600s limit
**Solution**: Implemented automatic timeout calculation based on file size:
- Files < 1GB: 1 hour timeout (default)
- Files > 1GB: Additional 30 minutes per GB
- Maximum: 4 hours
- Provides clear timeout error messages with hints for manual adjustment
**Status**: ✅ Fixed

### 6. Memory Usage with Batch Processing (FIXED)
**Issue**: Batch processing doesn't limit concurrent jobs, can exhaust memory
**Solution**: Added dynamic memory monitoring and automatic job limiting based on available RAM
**Details**: 
  - Checks available memory before starting batch jobs
  - Automatically reduces max_parallel based on available memory
  - Monitors memory during execution and waits if critically low (<1GB)
  - Default limit of 4 concurrent jobs, adjusted down if memory is limited
**Status**: ✅ Fixed

## Resolved Issues

### Hardware Acceleration and Timeout Improvements (2025-10-03)
- Enhanced NVENC detection to verify encoder availability in FFmpeg
- Added helpful warnings when GPU detected but encoder unavailable
- Implemented automatic timeout calculation based on file size
- Added progress monitoring with elapsed time and timeout display
- Better error messages for timeout failures with adjustment hints
- Improved test cleanup to prevent lingering processes

### Port Configuration (2025-09-17)
- Changed web interface default port from 8080 to 8098 to avoid conflicts
- Added FFMPEG_WEB_PORT and FFMPEG_API_PORT to defaults.sh
- Both ports now configurable via environment variables

### API Server Startup (2025-09-16)  
- Fixed Python API server not starting correctly
- Replaced deprecated cgi module with custom multipart parser
- Added proper PID management for server lifecycle

### Memory Management and Input Validation (2025-09-28)
- Added dynamic memory monitoring to batch processing
- Implemented automatic job limiting based on available RAM
- Added input file validation (already existed, documentation updated)
- Fixed error handling in API server (already existed, documentation updated)

## Recommended Improvements

1. **Add Job Queue System**
   - Integrate with PostgreSQL for persistent job storage
   - Add job priority and scheduling
   - Enable distributed processing

2. **Enhance Error Messages**
   - Add specific error codes for different failure types
   - Include troubleshooting hints in error responses
   - Log detailed errors for debugging

3. **Improve Resource Management**
   - Add memory usage monitoring
   - Limit concurrent processing jobs
   - Implement graceful degradation under load

4. **Add Progress Tracking**
   - WebSocket support for real-time progress updates
   - Store conversion history and statistics
   - Generate performance reports

5. **Security Enhancements**
   - Add rate limiting per IP
   - Implement API key authentication
   - Add file virus scanning before processing

## Testing Notes

- All smoke tests pass consistently
- Integration tests work with sample media files
- API endpoints respond correctly after port fix
- Hardware acceleration works when available
- Monitor functionality tracks resource usage properly