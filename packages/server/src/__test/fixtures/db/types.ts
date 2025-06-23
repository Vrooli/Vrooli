/**
 * Common types and interfaces for enhanced database fixtures
 * Inspired by the API fixtures system to provide better type safety,
 * organization, and comprehensive test coverage
 */

// Re-export types from Prisma for convenience
export type { Prisma, PrismaClient } from "@prisma/client";

/**
 * Generic database result type
 */
export type DbResult<T = any> = T & { id: string };

/**
 * Standard structure for database test fixtures
 * Provides consistent categories across all model fixtures
 */
export interface DbTestFixtures<TCreate, TUpdate = Partial<TCreate>> {
    /** Minimal valid data for basic testing */
    minimal: TCreate;
    /** Complete data with all optional fields for comprehensive testing */
    complete: TCreate;
    /** Invalid data scenarios for error testing */
    invalid: {
        /** Data missing required fields */
        missingRequired: Partial<TCreate>;
        /** Data with wrong types */
        invalidTypes: Record<string, any>;
        /** Additional invalid scenarios */
        [key: string]: any;
    };
    /** Edge cases and boundary conditions */
    edgeCases: {
        [key: string]: TCreate;
    };
    /** Update scenarios if different from create */
    updates?: {
        minimal: TUpdate;
        complete: TUpdate;
        [key: string]: TUpdate;
    };
}

/**
 * Configuration for enhanced database factories
 */
export interface DbFactoryConfig<TCreate> {
    /** Whether to include authentication */
    withAuth?: boolean;
    /** Team IDs to associate with */
    withTeams?: Array<{ teamId: string; role: string }>;
    /** Role IDs to assign */
    withRoles?: Array<{ id: string; name: string }>;
    /** Whether to create related objects */
    withRelations?: boolean;
    /** Field overrides */
    overrides?: Partial<TCreate>;
}

/**
 * Result of factory creation with metadata
 */
export interface DbFactoryResult<TCreate> {
    /** The created data */
    data: TCreate;
    /** Metadata about what was created */
    metadata: {
        hasAuth: boolean;
        teamCount: number;
        roleCount: number;
        relationCount: number;
    };
}

/**
 * Validation result for fixtures
 */
export interface DbFixtureValidation {
    /** Whether the fixture is valid */
    isValid: boolean;
    /** Validation errors if any */
    errors: string[];
    /** Warnings for potential issues */
    warnings: string[];
}

/**
 * Base interface for enhanced database factories
 */
export interface EnhancedDbFactory<TCreate, TUpdate = Partial<TCreate>> {
    /** Create minimal fixture */
    createMinimal(overrides?: Partial<TCreate>): TCreate;
    /** Create complete fixture */
    createComplete(overrides?: Partial<TCreate>): TCreate;
    /** Create fixture with automatic relationship setup */
    createWithRelationships(config: DbFactoryConfig<TCreate>): DbFactoryResult<TCreate>;
    /** Validate fixture data */
    validateFixture(data: TCreate): DbFixtureValidation;
    /** Create invalid fixture for error testing */
    createInvalid(scenario: string): any;
    /** Create edge case fixture */
    createEdgeCase(scenario: string): TCreate;
}

/**
 * Seeding options for bulk operations
 */
export interface BulkSeedOptions {
    /** Number of records to create */
    count?: number;
    /** Whether to include authentication */
    withAuth?: boolean;
    /** Whether to include bots */
    withBots?: boolean;
    /** Team ID for membership */
    teamId?: string;
    /** Custom overrides for each record */
    overrides?: Array<Record<string, any>>;
}

/**
 * Result of bulk seeding operations
 */
export interface BulkSeedResult<T> {
    /** Created records */
    records: T[];
    /** Summary statistics */
    summary: {
        total: number;
        withAuth: number;
        bots: number;
        teams: number;
        /** Additional fields for specific object types */
        lists?: number;        // For bookmark lists
        bookmarks?: number;    // For bookmarks in lists
        anonymous?: number;    // For anonymous views
        [key: string]: number | undefined; // Allow additional summary fields
    };
}

/**
 * Cross-layer integration types
 */
export interface ApiDbMapping<TApi, TDb> {
    /** Convert API fixture to DB fixture */
    apiToDb(apiData: TApi): TDb;
    /** Convert DB fixture to API format */
    dbToApi(dbData: TDb): TApi;
    /** Setup integrated test environment */
    setupIntegratedTest(scenario: string, options?: {
        seedDatabase?: boolean;
        mockApiResponses?: boolean;
    }): Promise<{ api: TApi; db: TDb }>;
}

/**
 * Error scenarios for systematic error testing
 */
export interface DbErrorScenarios {
    /** Constraint violation scenarios */
    constraints: {
        uniqueViolation: Record<string, any>;
        foreignKeyViolation: Record<string, any>;
        checkConstraintViolation: Record<string, any>;
    };
    /** Data validation scenarios */
    validation: {
        requiredFieldMissing: Record<string, any>;
        invalidDataType: Record<string, any>;
        outOfRange: Record<string, any>;
    };
    /** Business logic scenarios */
    businessLogic: {
        [key: string]: Record<string, any>;
    };
}

/**
 * Configuration fixtures for different test environments
 */
export interface DbConfigFixtures {
    /** Test database configurations */
    database: {
        minimal: Record<string, any>;
        complete: Record<string, any>;
    };
    /** Permission configurations */
    permissions: {
        [key: string]: Record<string, any>;
    };
    /** Feature flags for testing */
    features: {
        [key: string]: boolean;
    };
}

/**
 * Event fixtures for real-time testing
 */
export interface DbEventFixtures {
    /** Database change events */
    changes: {
        create: Record<string, any>;
        update: Record<string, any>;
        delete: Record<string, any>;
    };
    /** Trigger events */
    triggers: {
        [key: string]: Record<string, any>;
    };
}

/**
 * Helper type for fixture categories
 */
export type FixtureCategory = "minimal" | "complete" | "invalid" | "edgeCase" | "error" | "config" | "event";

/**
 * Utility type for extracting create input from Prisma models
 */
export type ExtractCreateInput<T> = T extends { create: (args: { data: infer U }) => any } ? U : never;

/**
 * Utility type for extracting update input from Prisma models
 */
export type ExtractUpdateInput<T> = T extends { update: (args: { data: infer U }) => any } ? U : never;

/**
 * Relationship configuration for database fixtures
 */
export interface RelationConfig {
    /** Field overrides for the main record */
    overrides?: Record<string, any>;
    /** Additional relationship-specific configurations */
    [key: string]: any;
}

/**
 * Relationship connections for existing records
 */
export interface RelationConnections {
    [key: string]: { id: string } | Array<{ id: string }>;
}

/**
 * Expected relationship counts for verification
 */
export interface RelationCounts {
    [key: string]: number;
}

/**
 * Test scenario configuration
 */
export interface TestScenario {
    /** Scenario name */
    name: string;
    /** Optional description */
    description?: string;
    /** Scenario configuration */
    config: Record<string, any>;
}

/**
 * Result of scenario execution
 */
export interface ScenarioResult {
    /** ID of the main record created */
    id: string;
    /** Scenario name */
    scenario: string;
    /** IDs of related records created */
    relatedIds: Record<string, string[]>;
}

/**
 * Database constraint validation result
 */
export interface ConstraintValidation {
    /** Whether all constraints are valid */
    valid: boolean;
    /** List of constraint violations */
    violations: string[];
}

/**
 * Base factory interface that all database fixtures implement
 */
export interface DatabaseFixtureFactory<TPrismaCreateInput, TPrismaInclude> {
    // Core factory methods
    createMinimal: (overrides?: Partial<TPrismaCreateInput>) => Promise<DbResult>;
    createComplete: (overrides?: Partial<TPrismaCreateInput>) => Promise<DbResult>;
    createWithRelations: (config: RelationConfig) => Promise<DbResult>;
    
    // Bulk operations
    seedMultiple: (count: number, template?: Partial<TPrismaCreateInput>) => Promise<DbResult[]>;
    seedScenario: (scenario: TestScenario | string) => Promise<ScenarioResult>;
    
    // Relationship management
    setupRelationships: (parentId: string, config: RelationConfig) => Promise<void>;
    connectExisting: (id: string, relations: RelationConnections) => Promise<void>;
    
    // Verification utilities
    verifyState: (id: string, expected: Partial<DbResult>) => Promise<void>;
    verifyRelationships: (id: string, expectedCounts: RelationCounts) => Promise<void>;
    verifyConstraints: (id: string) => Promise<ConstraintValidation>;
    
    // Cleanup utilities
    cleanup: (ids: string[]) => Promise<void>;
    cleanupAll: () => Promise<void>;
    cleanupRelated: (id: string, options?: { depth?: number; include?: string[] }) => Promise<void>;
    
    // Factory methods
    createFactory: (defaults?: Partial<TPrismaCreateInput>) => (overrides?: Partial<TPrismaCreateInput>) => Promise<DbResult>;
    validateCreate: (input: TPrismaCreateInput) => Promise<{ valid: boolean; errors: string[] }>;
    
    // Utility methods
    getCreatedIds: () => string[];
    clearTracking: () => void;
    getAvailableScenarios: () => string[];
}
