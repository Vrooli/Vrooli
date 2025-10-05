# Testing Guide - test-data-generator API

## Quick Reference

### Running Tests
```bash
# From api directory
npm test                 # Run all tests with coverage
npm run test:coverage    # Detailed coverage report
npm run test:watch       # Watch mode for development
npm run test:verbose     # Verbose output

# From scenario root
make test                # Run via Makefile

# Via test phases
./test/phases/test-unit.sh
```

## Test Suite Overview

### Coverage: 95.56%
- **48 test cases** covering all API endpoints
- **Statements**: 95.56%
- **Branches**: 90.00%
- **Functions**: 95.65%
- **Lines**: 96.71%

## Test Structure

### Test Categories

#### 1. Health & Configuration (3 tests)
- Health endpoint validation
- Data types listing
- Definition structures

#### 2. Data Generation (24 tests)
- **Users**: 12 tests (all fields, validation, seeding)
- **Companies**: 3 tests (structure, fields, seeding)
- **Products**: 3 tests (structure, numeric validation)
- **Orders**: 1 test (error handling for unimplemented)
- **Custom**: 5 tests (schema types, validation, seeding)

#### 3. Format Support (5 tests)
- JSON, XML, SQL, CSV formats
- Format-specific validation

#### 4. Error Handling (5 tests)
- 404 errors
- Invalid methods
- Malformed requests

#### 5. Edge Cases (9 tests)
- Boundary values (1, 10000)
- Invalid inputs
- Data consistency

#### 6. Performance (3 tests)
- Large dataset generation
- Concurrent requests

## Writing New Tests

### Test Pattern
```javascript
describe('Feature Name', () => {
  test('should do something specific', async () => {
    const response = await request(app)
      .post('/api/endpoint')
      .send({ data })
      .expect(200);

    expect(response.body.field).toBeDefined();
    expect(response.body.success).toBe(true);
  });
});
```

### Error Testing Pattern
```javascript
test('should return 400 for invalid input', async () => {
  const response = await request(app)
    .post('/api/endpoint')
    .send({ invalid: 'data' })
    .expect(400);

  expect(response.body.error).toBeDefined();
});
```

### Seed Testing Pattern
```javascript
test('should generate consistent data with seed', async () => {
  const requestBody = {
    count: 3,
    seed: '12345',
    fields: ['name', 'email'] // Exclude UUIDs
  };

  const response1 = await request(app)
    .post('/api/generate/users')
    .send(requestBody);

  const response2 = await request(app)
    .post('/api/generate/users')
    .send(requestBody);

  expect(response1.body.data[0].name).toEqual(response2.body.data[0].name);
});
```

## Common Assertions

### Response Structure
```javascript
// Success response
expect(response.body).toHaveProperty('success');
expect(response.body).toHaveProperty('type');
expect(response.body).toHaveProperty('count');
expect(response.body).toHaveProperty('timestamp');
expect(response.body).toHaveProperty('data');
expect(response.body).toHaveProperty('format');

// Error response
expect(response.body).toHaveProperty('error');
expect(response.body.success).toBe(false);
```

### Data Validation
```javascript
// Array validation
expect(Array.isArray(response.body.data)).toBe(true);
expect(response.body.data).toHaveLength(expectedCount);

// Type validation
expect(typeof item.price).toBe('number');
expect(typeof item.active).toBe('boolean');

// Field presence
expect(item.id).toBeDefined();
expect(item.email).toContain('@');
```

## Coverage Gaps

### Uncovered Lines (4 lines, 4.44%)
1. **Line 261**: Default case in formatData switch
2. **Line 324**: Development-only error message
3. **Lines 354-356**: Server startup logging (excluded in test mode)

### Why These Are Uncovered
- Line 261: Unreachable due to Joi validation
- Line 324: Requires NODE_ENV=development (tests use NODE_ENV=test)
- Lines 354-356: Server doesn't start in test mode

## Testing Best Practices

### ✅ Do
- Test both success and error paths
- Use descriptive test names
- Test edge cases (0, 1, max values)
- Validate response structure AND content
- Use seeds for deterministic tests (exclude UUIDs)
- Clean up resources in afterAll

### ❌ Don't
- Rely on test execution order
- Use hardcoded UUIDs in assertions
- Skip error case testing
- Test only happy paths
- Ignore performance implications

## Performance Guidelines

### Test Timeouts
- Default: 10 seconds (jest.config.js)
- Large datasets: 30 seconds (marked in test)
- CI/CD: Consider longer timeouts for 10k records

### Performance Expectations
- 100 records: < 5 seconds
- 1000 records: < 10 seconds
- 10000 records: < 30 seconds

## Debugging Failed Tests

### Common Issues

#### 1. Port Already in Use
```
Error: listen EADDRINUSE: address already in use :::PORT
```
**Solution**: Server already running. The test should use NODE_ENV=test to prevent server startup.

#### 2. Seed Inconsistency
```
Expected values to match but UUIDs differ
```
**Solution**: Exclude UUID fields from seed-based comparisons. UUIDs are not seeded by faker.

#### 3. Timeout Errors
```
Error: Timeout - Async callback was not invoked within the 10000 ms timeout
```
**Solution**: Increase timeout for large dataset tests using `jest.setTimeout()` or test-specific timeout.

## Integration with Vrooli Testing Infrastructure

### Test Phase Script
Location: `test/phases/test-unit.sh`

```bash
#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::unit::run_all_tests \
    --node-dir "api" \
    --skip-go \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

testing::phase::end_with_summary "Unit tests completed"
```

### Coverage Thresholds
- **Warning**: < 80% coverage
- **Error**: < 50% coverage
- **Current**: 95.56% coverage ✅

## Continuous Integration

### Pre-commit Checks
```bash
npm test  # Must pass before committing
```

### Coverage Requirements
All changes must maintain:
- Statement coverage ≥ 80%
- Branch coverage ≥ 80%
- Function coverage ≥ 80%
- Line coverage ≥ 80%

## Resources

### Documentation
- [Jest Documentation](https://jestjs.io/docs/getting-started)
- [Supertest Documentation](https://github.com/visionmedia/supertest)
- [Vrooli Testing Standards](/docs/testing/guides/scenario-unit-testing.md)

### Example Test Suites
- Gold Standard: `/scenarios/visited-tracker/` (79.4% Go coverage)
- Node.js Examples: Check other scenarios with Jest tests

## Test Maintenance

### When to Update Tests
1. Adding new endpoints
2. Changing API response structures
3. Modifying validation rules
4. Adding new data types or formats
5. Fixing bugs (add regression tests)

### Test Review Checklist
- [ ] All new code has corresponding tests
- [ ] Coverage remains above 80%
- [ ] All tests pass locally
- [ ] Performance tests updated if needed
- [ ] Error cases covered
- [ ] Documentation updated

## Support

For issues or questions:
1. Check this guide
2. Review existing tests for patterns
3. Consult Vrooli testing documentation
4. Review Jest/Supertest documentation
