/**
 * Conditional Database Test Example
 * 
 * This test demonstrates how to write tests that work both with and without database.
 * Tests gracefully skip database operations when testcontainers aren't available.
 */

import { describe, it, expect, beforeAll } from 'vitest';
import { getTestDatabaseClient, createTestTransaction } from '../../setup.vitest.js';
import { BookmarkFixtureFactory } from '../factories/BookmarkFixtureFactory.js';

describe('Bookmark Tests - Works With or Without Database', () => {
    let factory: BookmarkFixtureFactory;
    let hasDatabase = false;

    beforeAll(async () => {
        factory = new BookmarkFixtureFactory();
        
        // Check if database is available
        const prisma = await getTestDatabaseClient();
        hasDatabase = !!prisma;
        
        console.log(`Running tests ${hasDatabase ? 'WITH' : 'WITHOUT'} database`);
    });

    it('should validate bookmark data (no database needed)', async () => {
        // This test uses only the fixture factory - no database required
        const validData = factory.createFormData('complete');
        const invalidData = factory.createFormData('invalid');
        
        const validResult = await factory.validateFormData(validData);
        const invalidResult = await factory.validateFormData(invalidData);
        
        expect(validResult.isValid).toBe(true);
        expect(invalidResult.isValid).toBe(false);
    });

    it('should transform form data to API format (no database needed)', () => {
        // This test uses only pure functions - no database required
        const formData = factory.createFormData('withNewList');
        const apiInput = factory.transformToAPIInput(formData);
        
        expect(apiInput.bookmarkFor).toBe(formData.bookmarkFor);
        expect(apiInput.forConnect).toBe(formData.forConnect);
        expect(apiInput.id).toMatch(/^test_\d+_[a-z0-9]+$/);
        
        if (formData.createNewList) {
            expect(apiInput.list.create.label).toBe(formData.newListLabel);
        }
    });

    it('should generate MSW handlers (no database needed)', () => {
        // This test creates mock handlers - no database required
        const handlers = factory.createMSWHandlers();
        
        expect(handlers.success).toBeDefined();
        expect(handlers.error).toBeDefined();
        expect(handlers.loading).toBeDefined();
        expect(handlers.networkError).toBeDefined();
    });

    it('should test database operations when available', async () => {
        const prisma = await getTestDatabaseClient();
        
        if (!prisma) {
            console.log('Skipping database test - testcontainers not available');
            return;
        }

        // Database is available, run integration test
        await createTestTransaction(async (tx) => {
            // Create test user
            const user = await tx.user.create({
                data: {
                    id: `test_user_${Date.now()}`,
                    handle: 'testuser',
                    email: 'test@example.com',
                    name: 'Test User'
                }
            });

            // Create bookmark list
            const list = await tx.bookmark_list.create({
                data: {
                    id: `test_list_${Date.now()}`,
                    created_by: user.id,
                    label: 'Test Bookmarks'
                }
            });

            // Verify creation
            expect(list.id).toBeDefined();
            expect(list.label).toBe('Test Bookmarks');
            expect(list.created_by).toBe(user.id);

            // Query to verify
            const found = await tx.bookmark_list.findUnique({
                where: { id: list.id }
            });

            expect(found).toBeDefined();
            expect(found!.id).toBe(list.id);
        });
    });

    it('should handle complex scenarios based on database availability', async () => {
        const scenarios = factory.createTestCases();
        
        for (const scenario of scenarios.slice(0, 3)) {
            // Always validate data structure
            const validation = await factory.validateFormData(scenario.formData);
            
            if (scenario.shouldSucceed) {
                expect(validation.isValid).toBe(true);
            } else {
                expect(validation.isValid).toBe(false);
            }
            
            // Only test database operations if available
            if (hasDatabase) {
                const prisma = await getTestDatabaseClient();
                
                await createTestTransaction(async (tx) => {
                    // Would create actual database records here
                    console.log(`Would test scenario: ${scenario.description} in database`);
                });
            } else {
                // Test with mocks only
                const mockResponse = factory.createMockResponse();
                expect(mockResponse).toBeDefined();
                expect(mockResponse.id).toBeDefined();
            }
        }
    });
});

describe('Performance Tests', () => {
    let factory: BookmarkFixtureFactory;

    beforeAll(() => {
        factory = new BookmarkFixtureFactory();
    });

    it('should generate fixtures efficiently', () => {
        const startTime = performance.now();
        
        // Generate 100 fixtures
        for (let i = 0; i < 100; i++) {
            factory.createFormData('complete');
            factory.createMockResponse();
            factory.state.factory.createUIState('success');
        }
        
        const duration = performance.now() - startTime;
        
        // Should complete quickly even without database
        expect(duration).toBeLessThan(500); // 500ms for 100 fixtures
    });

    it('should validate data efficiently', async () => {
        const formData = factory.createFormData('complete');
        const startTime = performance.now();
        
        // Validate 50 times
        const validations = await Promise.all(
            Array(50).fill(null).map(() => factory.validateFormData(formData))
        );
        
        const duration = performance.now() - startTime;
        
        expect(validations.every(v => v.isValid)).toBe(true);
        expect(duration).toBeLessThan(200); // 200ms for 50 validations
    });
});

/**
 * This example demonstrates:
 * 
 * 1. ✅ Tests that work with or without database
 * 2. ✅ Graceful handling when testcontainers aren't available
 * 3. ✅ Mix of unit tests (no DB) and integration tests (with DB)
 * 4. ✅ Performance testing without database dependency
 * 5. ✅ Conditional test execution based on environment
 * 6. ✅ Same test file can run in both modes
 * 
 * Key patterns:
 * - Check database availability with getTestDatabaseClient()
 * - Skip database operations when not available
 * - Always test what you can without database
 * - Use factory methods for consistent test data
 */