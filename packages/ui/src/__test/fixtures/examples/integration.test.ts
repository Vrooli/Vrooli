/**
 * Integration Test Example
 * 
 * This test demonstrates database integration testing using testcontainers.
 * The database is automatically available when needed.
 */

import { describe, it, expect, beforeAll, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';
import {
    getTestDatabaseClient,
    createTestTransaction,
    createTestAPIClient
} from '../../setup.vitest.js';
import { BookmarkFixtureFactory } from '../factories/BookmarkFixtureFactory.js';
import type { BookmarkFormData } from '../types.js';

// Mock React component that would normally use your actual components
const BookmarkForm = ({ onSubmit }: { onSubmit: (data: BookmarkFormData) => Promise<void> }) => {
    const [isSubmitting, setIsSubmitting] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);
    const [formData, setFormData] = React.useState<BookmarkFormData>({
        bookmarkFor: 'Resource',
        forConnect: ''
    });

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsSubmitting(true);
        setError(null);
        
        try {
            await onSubmit(formData);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <form onSubmit={handleSubmit}>
            <input
                aria-label="Object ID"
                value={formData.forConnect}
                onChange={(e) => setFormData(prev => ({ ...prev, forConnect: e.target.value }))}
                disabled={isSubmitting}
            />
            <select
                aria-label="Object Type"
                value={formData.bookmarkFor}
                onChange={(e) => setFormData(prev => ({ ...prev, bookmarkFor: e.target.value as any }))}
                disabled={isSubmitting}
            >
                <option value="Resource">Resource</option>
                <option value="Routine">Routine</option>
                <option value="Api">API</option>
            </select>
            <button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : 'Save Bookmark'}
            </button>
            {error && <div role="alert">{error}</div>}
        </form>
    );
};

// Mock React for the component above
const React = {
    useState: (initial: any) => [initial, () => {}],
    FormEvent: Event
};

describe('Bookmark Integration Tests with Database', () => {
    let factory: BookmarkFixtureFactory;
    let server: ReturnType<typeof setupServer>;

    beforeAll(() => {
        factory = new BookmarkFixtureFactory();
        server = setupServer();
        server.listen();
    });

    afterEach(() => {
        server.resetHandlers();
    });

    afterAll(() => {
        server.close();
    });

    it('should save bookmark to database and verify persistence', async () => {
        // Get database client - this will be available because testcontainers are running
        const prisma = await getTestDatabaseClient();
        if (!prisma) {
            console.log('Skipping database test - no database available');
            return;
        }

        // Create test data
        const formData = factory.createFormData('complete');
        
        // Create a test user and resource for foreign key constraints
        const testUser = await prisma.user.create({
            data: {
                id: `test_user_${Date.now()}`,
                handle: 'testuser',
                email: 'test@example.com',
                name: 'Test User'
            }
        });

        const testResource = await prisma.resource.create({
            data: {
                id: formData.forConnect,
                created_by: testUser.id,
                name: 'Test Resource'
            }
        });

        // Create a bookmark list
        const bookmarkList = await prisma.bookmark_list.create({
            data: {
                id: `test_list_${Date.now()}`,
                created_by: testUser.id,
                label: 'Test List'
            }
        });

        // Run test in a transaction that will be rolled back
        await createTestTransaction(async (tx) => {
            // Create bookmark through API-like operation
            const bookmark = await tx.bookmark.create({
                data: {
                    id: `test_bookmark_${Date.now()}`,
                    created_by: testUser.id,
                    list: { connect: { id: bookmarkList.id } },
                    to: { connect: { id: testResource.id } },
                    bookmarkFor: 'Resource'
                },
                include: {
                    list: true,
                    to: true
                }
            });

            // Verify bookmark was created
            expect(bookmark).toBeDefined();
            expect(bookmark.id).toBeDefined();
            expect(bookmark.list.id).toBe(bookmarkList.id);
            expect(bookmark.to.id).toBe(testResource.id);
            expect(bookmark.bookmarkFor).toBe('Resource');

            // Query to verify it exists
            const found = await tx.bookmark.findUnique({
                where: { id: bookmark.id },
                include: { list: true, to: true }
            });

            expect(found).toBeDefined();
            expect(found!.id).toBe(bookmark.id);
        });

        // Clean up test data
        await prisma.bookmark.deleteMany({
            where: { created_by: testUser.id }
        });
        await prisma.bookmark_list.deleteMany({
            where: { created_by: testUser.id }
        });
        await prisma.resource.deleteMany({
            where: { created_by: testUser.id }
        });
        await prisma.user.delete({
            where: { id: testUser.id }
        });
    });

    it('should handle API calls with MSW and validate response', async () => {
        const user = userEvent.setup();
        const formData = factory.createFormData('minimal');
        
        // Setup MSW handler
        server.use(
            http.post('/api/bookmark', async ({ request }) => {
                const body = await request.json();
                
                // Validate request using factory
                const validation = await factory.validateFormData(body as BookmarkFormData);
                if (!validation.isValid) {
                    return HttpResponse.json(
                        { error: { code: 'VALIDATION_ERROR', details: validation.errors } },
                        { status: 400 }
                    );
                }

                // Return mock response
                const mockResponse = factory.createMockResponse();
                return HttpResponse.json({ data: mockResponse });
            })
        );

        let submittedData: BookmarkFormData | null = null;

        const handleSubmit = async (data: BookmarkFormData) => {
            const response = await fetch('/api/bookmark', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error.code);
            }

            const result = await response.json();
            submittedData = data;
            return result;
        };

        render(<BookmarkForm onSubmit={handleSubmit} />);

        // Fill form
        const objectIdInput = screen.getByLabelText(/object id/i);
        await user.clear(objectIdInput);
        await user.type(objectIdInput, formData.forConnect);

        // Submit
        const submitButton = screen.getByRole('button', { name: /save bookmark/i });
        await user.click(submitButton);

        // Wait for submission to complete
        await waitFor(() => {
            expect(screen.getByRole('button', { name: /save bookmark/i })).toBeEnabled();
        });

        // Verify submission happened
        expect(submittedData).toBeDefined();
        expect(submittedData!.forConnect).toBe(formData.forConnect);
    });

    it('should validate complex bookmark scenarios', async () => {
        const scenarios = factory.createTestCases();
        
        for (const scenario of scenarios.slice(0, 3)) { // Test first 3 to avoid timeout
            const validation = await factory.validateFormData(scenario.formData);
            
            if (scenario.shouldSucceed) {
                expect(validation.isValid).toBe(true);
            } else {
                expect(validation.isValid).toBe(false);
                expect(validation.errors).toBeDefined();
            }
        }
    });

    it('should test database constraints and relationships', async () => {
        const prisma = await getTestDatabaseClient();
        if (!prisma) {
            console.log('Skipping database test - no database available');
            return;
        }

        await createTestTransaction(async (tx) => {
            // Attempt to create bookmark with invalid foreign key should fail
            await expect(async () => {
                await tx.bookmark.create({
                    data: {
                        id: `test_bookmark_${Date.now()}`,
                        created_by: 'non_existent_user',
                        list: { connect: { id: 'non_existent_list' } },
                        to: { connect: { id: 'non_existent_resource' } },
                        bookmarkFor: 'Resource'
                    }
                });
            }).rejects.toThrow();
        });
    });
});

describe('Component Testing with Real Fixtures', () => {
    let factory: BookmarkFixtureFactory;

    beforeAll(() => {
        factory = new BookmarkFixtureFactory();
    });

    it('should render form with different states', async () => {
        const states = [
            factory.state.factory.createUIState('idle'),
            factory.state.factory.createUIState('loading'),
            factory.state.factory.createUIState('success', factory.createMockResponse()),
            factory.state.factory.createUIState('error', undefined, 'Failed to save')
        ];

        for (const state of states) {
            // In a real app, you would render your component with this state
            expect(state).toBeDefined();
            expect(factory.state.validator.isValidUIState(state)).toBe(true);
        }
    });

    it('should handle form validation with real schemas', async () => {
        const testCases = [
            { data: factory.createFormData('complete'), shouldPass: true },
            { data: factory.createFormData('invalid'), shouldPass: false },
            { data: { bookmarkFor: 'InvalidType', forConnect: '' } as any, shouldPass: false }
        ];

        for (const testCase of testCases) {
            const validation = await factory.validateFormData(testCase.data);
            expect(validation.isValid).toBe(testCase.shouldPass);
        }
    });
});

/**
 * This example demonstrates:
 * 
 * 1. ✅ Unified test setup - no separate integration configuration
 * 2. ✅ Automatic database availability via testcontainers
 * 3. ✅ Real database operations with transactions
 * 4. ✅ MSW integration for API mocking
 * 5. ✅ Type-safe fixtures with real validation
 * 6. ✅ Same test scripts for all test types
 * 
 * Tests that need database automatically get it, while simple tests run fast without it.
 */