# Study Buddy - Known Issues & Problems

## Critical Issues (P0)

### 1. Database Schema Not Applied ❌
**Problem**: The PostgreSQL schema is not being applied to the database
- **Impact**: Core functionality broken - cannot create study sessions, subjects, or flashcards
- **Error**: `pq: relation "study_sessions" does not exist`
- **Root Cause**: The schema.sql file exists but is not executed during resource population
- **Workaround**: Manually apply schema using resource-postgres CLI
- **Fix Required**: Update population script to properly apply schema

### 2. N8N Workflows Not Loaded ⚠️
**Problem**: N8N workflows fail to load during resource population
- **Impact**: AI features partially working (fallback to mock data)
- **Evidence**: Population logs show 10 workflow failures
- **Files Affected**: All workflows in `/initialization/n8n/`
- **Workaround**: API has fallback implementations for critical features

## Moderate Issues (P1)

### 3. CLI Response Parsing Issues ⚠️
**Problem**: CLI expects different response format than API provides
- **Impact**: CLI commands fail even when API succeeds
- **Example**: `generate-flashcards` expects `"success":true` but API returns `"cards"`
- **Files**: `/cli/study-buddy` lines 86-92
- **Fix**: Update CLI to match actual API response format

### 4. Spaced Repetition Not Persisted 🔄
**Problem**: Spaced repetition calculations work but don't persist to database
- **Impact**: Algorithm resets between sessions
- **Cause**: Database schema issues prevent persistence
- **Workaround**: In-memory calculation works for single session

## Minor Issues (P2)

### 5. Test Infrastructure Incomplete 📝
**Problem**: Legacy test format, missing unit tests
- **Impact**: Reduced confidence in changes
- **Status**: Basic test structure created, needs expansion
- **Recommendation**: Add unit tests for Go API and UI components

### 6. Port Configuration Inconsistency 🔌
**Problem**: Multiple port values in different places
- **Files**: `.env`, runtime configs, test defaults
- **Impact**: Tests may fail if ports change
- **Fix Applied**: Tests now dynamically detect ports

## Validation Results

### Working Features ✅
- ✅ API health endpoint
- ✅ Flashcard generation (with Ollama fallback)
- ✅ Due cards retrieval (mock data)
- ✅ XP calculation and progress tracking (in-memory)
- ✅ UI loads with cozy aesthetic
- ✅ CLI help and basic commands

### Partially Working ⚠️
- ⚠️ Study sessions (API works, no persistence)
- ⚠️ Subject management (API works, no persistence)
- ⚠️ Spaced repetition (algorithm works, no persistence)

### Not Working ❌
- ❌ Data persistence to PostgreSQL
- ❌ N8N workflow integration
- ❌ CLI flashcard generation
- ❌ Real user authentication

## Next Steps

1. **Immediate**: Fix database schema application
2. **Short-term**: Update CLI to match API responses
3. **Medium-term**: Implement proper N8N workflow loading
4. **Long-term**: Add comprehensive test coverage

## Notes for Future Improvers

- The API has good fallback mechanisms for when resources aren't available
- The UI is well-designed but lacks connection to real data
- The spaced repetition algorithm (SM-2) is properly implemented
- Most P0 features exist in code but fail due to infrastructure issues

---

*Last Updated: 2025-09-30*
*Identified During: Ecosystem Improver Task*