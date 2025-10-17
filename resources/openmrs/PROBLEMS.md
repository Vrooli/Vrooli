# Known Issues and Solutions

## 1. REST API Not Available on First Run
**Problem**: The REST API returns 302 redirects to `/openmrs/initialsetup` on first run.

**Root Cause**: OpenMRS requires initial configuration through the web interface before the REST API becomes available.

**Solution**: 
1. Start OpenMRS with `vrooli resource openmrs manage start --wait`
2. Navigate to http://localhost:8005/openmrs
3. Complete the initial setup wizard
4. The REST API will then be available at http://localhost:8005/openmrs/ws/rest/v1

**Future Improvement**: Automate the initial setup process using environment variables or a setup script.

## 2. PostgreSQL Not Supported
**Problem**: Initial implementation attempted to use PostgreSQL, but OpenMRS requires MySQL.

**Root Cause**: OpenMRS has hardcoded MySQL-specific queries and doesn't support PostgreSQL.

**Solution**: Switched to MySQL 5.7 which is fully supported by OpenMRS.

## 3. Long Startup Time
**Problem**: OpenMRS takes 2-3 minutes to start on first run.

**Root Cause**: 
- Large Docker image download (300+ MB)
- Database initialization
- Module loading

**Solution**: Use the `--wait` flag when starting to wait for health checks.

## 4. API Authentication
**Problem**: Some API endpoints require session authentication beyond basic auth.

**Root Cause**: OpenMRS uses a session-based authentication system.

**Future Improvement**: Implement proper session management in the CLI utilities.