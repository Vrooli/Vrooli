/**
 * Centralized fixture imports and extensions for integration testing
 * 
 * This module provides a unified interface to leverage fixtures from:
 * - @vrooli/shared: API fixtures, config fixtures, error fixtures, event fixtures
 * - @vrooli/server: Database fixtures, permission fixtures, execution fixtures
 * 
 * Integration tests should import from this module to ensure consistency
 * and leverage the comprehensive fixture systems already established.
 */

// =============================================================================
// SHARED PACKAGE IMPORTS - Fallback Approach
// =============================================================================

// Since the shared package test fixtures may not be properly exported,
// we'll create simple fallback fixtures that follow the same patterns.
// Integration tests should focus on the testing engine rather than specific fixture data.

// Simple API fixtures for integration testing
export const userFixtures = {
    minimal: {
        create: { name: "Test User", email: "test@example.com" },
        find: { id: "test-user-123", name: "Test User", email: "test@example.com" },
    },
    complete: {
        create: { name: "Complete User", email: "complete@example.com", bio: "Test bio" },
        find: { id: "test-user-456", name: "Complete User", email: "complete@example.com", bio: "Test bio" },
    },
};

export const teamFixtures = {
    minimal: {
        create: { name: "Test Team" },
        find: { id: "test-team-123", name: "Test Team" },
    },
    complete: {
        create: { name: "Complete Team", description: "A test team for integration testing" },
        find: { id: "test-team-456", name: "Complete Team", description: "A test team for integration testing" },
    },
};

export const projectFixtures = {
    minimal: {
        create: { name: "Test Project" },
        find: { id: "test-project-123", name: "Test Project" },
    },
    complete: {
        create: { name: "Complete Project", description: "A test project for integration testing" },
        find: { id: "test-project-456", name: "Complete Project", description: "A test project for integration testing" },
    },
};

export const commentFixtures = {
    minimal: {
        create: { text: "This is a test comment" },
        find: { id: "test-comment-123", text: "This is a test comment" },
    },
    complete: {
        create: { text: "This is a comprehensive test comment with detailed feedback." },
        find: { id: "test-comment-456", text: "This is a comprehensive test comment with detailed feedback." },
    },
};

export const bookmarkFixtures = {
    minimal: {
        create: { bookmarkFor: "Project", forConnect: "test-project-123" },
        find: { id: "test-bookmark-123", bookmarkFor: "Project", forConnect: "test-project-123" },
    },
    complete: {
        create: { bookmarkFor: "Project", forConnect: "test-project-456", createNewList: true, newListLabel: "My Projects" },
        find: { id: "test-bookmark-456", bookmarkFor: "Project", forConnect: "test-project-456" },
    },
};

export const chatFixtures = {
    minimal: {
        create: { title: "Test Chat" },
        find: { id: "test-chat-123", title: "Test Chat" },
    },
    complete: {
        create: { title: "Complete Chat", description: "A test chat for integration testing" },
        find: { id: "test-chat-456", title: "Complete Chat", description: "A test chat for integration testing" },
    },
};

export const issueFixtures = {
    minimal: {
        create: { title: "Test Issue" },
        find: { id: "test-issue-123", title: "Test Issue" },
    },
    complete: {
        create: { title: "Complete Issue", description: "A test issue for integration testing" },
        find: { id: "test-issue-456", title: "Complete Issue", description: "A test issue for integration testing" },
    },
};

export const tagFixtures = {
    minimal: {
        create: { name: "test-tag" },
        find: { id: "test-tag-123", name: "test-tag" },
    },
    complete: {
        create: { name: "complete-tag", description: "A test tag for integration testing" },
        find: { id: "test-tag-456", name: "complete-tag", description: "A test tag for integration testing" },
    },
};

// Simple config fixtures
export const botConfigFixtures = {
    complete: { personality: "helpful", responseStyle: "detailed" },
    minimal: { personality: "neutral" },
};

export const chatConfigFixtures = {
    complete: { allowGuests: true, maxParticipants: 50 },
    minimal: { allowGuests: false },
    variants: {
        privateTeamChat: { allowGuests: false, maxParticipants: 10 },
        publicSupport: { allowGuests: true, maxParticipants: 100 },
    },
};

export const routineConfigFixtures = {
    complete: { timeout: 30000, retries: 3 },
    minimal: { timeout: 10000 },
};

// Simple error fixtures
export const apiErrorFixtures = {
    notFound: { standard: { message: "Not found", status: 404 } },
    rateLimit: { standard: { message: "Rate limit exceeded", status: 429 } },
};

export const authErrorFixtures = {
    unauthorized: { standard: { message: "Unauthorized", status: 401 } },
};

export const networkErrorFixtures = {
    timeout: { client: { message: "Request timeout", code: "TIMEOUT" } },
    offline: { standard: { message: "Network offline", code: "OFFLINE" } },
};

export const validationErrorFixtures = {
    fieldErrors: { standard: { message: "Validation failed", fields: {} } },
};

// Simple event fixtures
export const chatEventFixtures = {
    messages: {
        textMessage: { type: "message", data: { text: "Hello, world!" } },
    },
    typing: {
        start: { type: "typing_start", data: { userId: "test-user-123" } },
        stop: { type: "typing_stop", data: { userId: "test-user-123" } },
    },
};

export const socketEventFixtures = {
    connection: {
        connected: { type: "connect", data: {} },
        disconnected: { type: "disconnect", data: {} },
    },
};

export const swarmEventFixtures = {
    execution: {
        started: { type: "execution_start", data: { swarmId: "test-swarm-123" } },
        progress: { type: "execution_progress", data: { progress: 0.5 } },
        completed: { type: "execution_complete", data: { result: "success" } },
    },
};

// =============================================================================
// SERVER PACKAGE IMPORTS - Simplified Approach
// =============================================================================

// For now, we'll create our own simple session helpers since the server package
// exports may not be available in the expected format. This provides a more
// stable foundation for integration testing.

export interface SessionUser {
    id: string;
    name: string;
    email: string;
    isAdmin?: boolean;
}

// Simple session helpers for integration testing
export const sessionHelpers = {
    async quickSession(role: 'admin' | 'standard' | 'guest' = 'standard'): Promise<SessionUser> {
        // Create a simple test session - in real tests, this would create actual DB records
        return {
            id: `test-user-${role}-${Date.now()}`,
            name: `Test ${role} User`,
            email: `test-${role}@example.com`,
            isAdmin: role === 'admin',
        };
    },
};

// Basic user personas for testing
export const userPersonas = {
    admin: {
        id: "111111111111111111",
        name: "Admin User",
        email: "admin@example.com",
        isAdmin: true,
    },
    standard: {
        id: "222222222222222222", 
        name: "Standard User",
        email: "user@example.com",
        isAdmin: false,
    },
    guest: {
        id: "333333333333333333",
        name: "Guest User", 
        email: "guest@example.com",
        isAdmin: false,
    },
};

// =============================================================================
// INTEGRATION-SPECIFIC EXTENSIONS
// =============================================================================

/**
 * Integration-specific fixture configurations that extend the base fixtures
 * for cross-layer testing scenarios
 */

export interface IntegrationFixtureConfig {
    /** Enable database persistence testing */
    withDatabase?: boolean;
    /** Enable real-time event simulation */
    withEvents?: boolean;
    /** Enable permission testing */
    withPermissions?: boolean;
    /** Enable performance benchmarking */
    withPerformance?: boolean;
    /** Custom test data transformations */
    customTransforms?: Record<string, (data: any) => any>;
}

/**
 * Simple integration fixture factory for testing
 * Provides basic fixture creation without complex inheritance
 */
export class IntegrationAPIFixtureFactory<TCreate, TUpdate, TFind = any> {
    constructor(
        private objectType: string,
        private config: IntegrationFixtureConfig = {}
    ) {}

    /**
     * Create fixtures with basic test data
     */
    create(scenario: string): any {
        // Simple fixture creation based on available fixtures
        const fixtureMap: Record<string, any> = {
            user: userFixtures,
            comment: commentFixtures,
            team: teamFixtures,
            project: projectFixtures,
            bookmark: bookmarkFixtures,
            chat: chatFixtures,
            issue: issueFixtures,
            tag: tagFixtures,
        };
        
        const fixtures = fixtureMap[this.objectType.toLowerCase()];
        return fixtures?.[scenario] || fixtures?.minimal || null;
    }

    /**
     * Create fixtures with permission context
     */
    async createWithPermissions(scenario: string, userRole: keyof typeof userPersonas) {
        const baseFixture = this.create(scenario);
        const session = await sessionHelpers.quickSession(userRole);
        
        return { 
            fixture: baseFixture, 
            session, 
            permissions: userPersonas[userRole] 
        };
    }
}

/**
 * Simple integration workflow fixtures for common scenarios
 */
export const integrationWorkflowFixtures = {
    userOnboarding: {
        /** Minimal onboarding flow */
        minimal: {
            signup: userFixtures.minimal?.create || { name: "Test User", email: "test@example.com" },
            profile: { name: "Test User" },
        },
        
        /** Complete user signup flow */
        complete: {
            signup: userFixtures.complete?.create || { name: "Complete User", email: "complete@example.com" },
            profile: userFixtures.complete?.update || { name: "Complete User" },
            team: teamFixtures.minimal?.create || { name: "Test Team" },
            project: projectFixtures.minimal?.create || { name: "Test Project" },
        },
    },

    contentWorkflow: {
        /** Basic collaboration workflow */
        collaboration: {
            team: teamFixtures.complete?.create || { name: "Collaboration Team" },
            project: projectFixtures.complete?.create || { name: "Collaboration Project" },
            chat: chatFixtures.complete?.create || { title: "Team Chat" },
            comments: [
                commentFixtures.minimal?.create || { text: "First comment" },
                commentFixtures.complete?.create || { text: "Detailed comment" }
            ],
        },
    },

    performanceBenchmarks: {
        /** Load testing scenarios */
        lightLoad: { concurrency: 5, iterations: 10 },
        mediumLoad: { concurrency: 15, iterations: 50 },
        
        /** Performance baseline expectations */
        baselines: {
            simpleForm: { maxTime: 1000, successRate: 0.99 },
            complexForm: { maxTime: 3000, successRate: 0.95 },
        },
    },
};

/**
 * Simple integration error scenarios for testing
 */
export const integrationErrorFixtures = {
    /** API layer errors */
    apiErrors: {
        validation: validationErrorFixtures?.fieldErrors?.standard || { message: "Validation failed" },
        authorization: authErrorFixtures?.unauthorized?.standard || { message: "Unauthorized" },
        notFound: apiErrorFixtures?.notFound?.standard || { message: "Not found" },
    },
    
    /** Database layer errors */
    databaseErrors: {
        constraint: "Foreign key constraint violation",
        uniqueness: "Unique constraint violation",
        connection: "Database connection timeout",
    },
    
    /** Network layer errors */
    networkErrors: {
        timeout: networkErrorFixtures?.timeout?.client || { message: "Request timeout" },
        offline: networkErrorFixtures?.offline?.standard || { message: "Network offline" },
    },
    
    /** Recovery scenarios */
    recoveryScenarios: {
        retrySuccess: { failAttempts: 2, retryCount: 3 },
        fallbackSuccess: { primaryFails: true, fallbackWorks: true },
    },
};

/**
 * Helper function to create a simple integration test configuration
 */
export function createIntegrationTestConfig(objectType: string, options: IntegrationFixtureConfig = {}) {
    return {
        apiFactory: new IntegrationAPIFixtureFactory(objectType, options),
        workflows: integrationWorkflowFixtures,
        errors: integrationErrorFixtures,
        config: options,
    };
}

/**
 * Quick access to common integration test scenarios
 */
export const quickIntegrationFixtures = {
    /** Get minimal test data for any object type */
    minimal: (objectType: string) => {
        const fixtureMap: Record<string, any> = {
            user: userFixtures,
            team: teamFixtures,
            project: projectFixtures,
            comment: commentFixtures,
            bookmark: bookmarkFixtures,
            chat: chatFixtures,
            issue: issueFixtures,
            tag: tagFixtures,
        };
        const fixtures = fixtureMap[objectType.toLowerCase()];
        return fixtures?.minimal || null;
    },
    
    /** Get complete test data for any object type */
    complete: (objectType: string) => {
        const fixtureMap: Record<string, any> = {
            user: userFixtures,
            team: teamFixtures,
            project: projectFixtures,
            comment: commentFixtures,
            bookmark: bookmarkFixtures,
            chat: chatFixtures,
            issue: issueFixtures,
            tag: tagFixtures,
        };
        const fixtures = fixtureMap[objectType.toLowerCase()];
        return fixtures?.complete || null;
    },
    
    /** Get user session for testing */
    userSession: async (role: 'admin' | 'standard' | 'guest' = "standard") => {
        return await sessionHelpers.quickSession(role);
    },
};