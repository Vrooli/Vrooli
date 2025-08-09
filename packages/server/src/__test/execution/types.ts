/**
 * Execution Test Framework Types
 * 
 * Comprehensive type definitions for the execution test framework
 */

// Import Session from @vrooli/shared (GraphQL type), not from server types
// This is used for test scenarios that simulate GraphQL responses
import type { Session } from "@vrooli/shared";

// Base test entity types
export interface TestEntity {
    id: string;
    created_at: Date;
    updated_at: Date;
    created_by: string;
}

// Routine types
export interface RoutineTestData extends TestEntity {
    __typename: "Routine";
    name: string;
    description: string;
    versions: RoutineVersionTestData[];
}

export interface RoutineVersionTestData {
    __typename: "RoutineVersion";
    id: string;
    routineId: string;
    version: string;
    isLatest: boolean;
    isPublic: boolean;
    inputs: RoutineInputTestData[];
    outputs: RoutineOutputTestData[];
    nodes: NodeTestData[];
    complexity: number;
    created_at: Date;
    updated_at: Date;
}

export interface RoutineInputTestData {
    __typename: "RoutineVersionInput";
    id: string;
    index: number;
    name: string;
    type: string;
    required: boolean;
    default?: unknown;
    routineVersionId: string;
}

export interface RoutineOutputTestData {
    __typename: "RoutineVersionOutput";
    id: string;
    index: number;
    name: string;
    type: string;
    routineVersionId: string;
}

export interface NodeTestData {
    __typename: "Node";
    id: string;
    type: string;
    index: number;
    routineVersionId: string;
    data: Record<string, unknown>;
}

// Agent types
export interface AgentTestData extends TestEntity {
    __typename: "Agent";
    name: string;
    goal: string;
    subscriptions: string[];
    behaviors: AgentBehaviorTestData[];
    resources?: AgentResourceConfig;
    teamId?: string;
}

export interface AgentBehaviorTestData {
    id: string;
    trigger: {
        topic: string;
        conditions?: Record<string, unknown>;
    };
    action: {
        id: string;
        type: "routine" | "emit" | "accumulate" | "decision";
        label?: string;
        topic?: string;
        outputOperations?: OutputOperations;
    };
}

export interface OutputOperations {
    set?: Array<{
        routineOutput?: string;
        value?: unknown;
        blackboardId: string;
    }>;
    append?: Array<{
        routineOutput?: string;
        value?: unknown;
        blackboardId: string;
    }>;
}

export interface AgentResourceConfig {
    maxCredits?: string;
    maxDurationMs?: number;
    preferredModel?: string;
}

// Team/Swarm types
export interface TeamTestData extends TestEntity {
    __typename: "Team";
    name: string;
    description: string;
    businessPrompt: string;
    details?: string;
    profitTarget?: string;
    agents: string[];
    resources: TeamResourceConfig;
}

export interface TeamResourceConfig {
    maxCredits: string;
    maxDurationMs: number;
    maxConcurrentAgents?: number;
}

// Database operation types
export interface DatabaseOperationConfig {
    session: Session;
    data: Record<string, unknown>;
}

export interface DatabaseUpdateConfig {
    session: Session;
    where: { id: string };
    data: Record<string, unknown>;
}

export interface DatabaseDeleteConfig {
    session: Session;
    where: { id: string };
}

// Mock response types
export interface MockResponse {
    attempt?: number;
    input?: Record<string, unknown>;
    response: Record<string, unknown>;
    delay?: number;
    error?: Error;
}

export interface MockRoutineResponse {
    attempt?: number;
    success: boolean;
    output?: Record<string, unknown>;
    logs?: string;
    error?: string;
}

export interface MockAIResponse {
    attempt?: number;
    input?: Record<string, unknown>;
    response: Record<string, unknown>;
    delay?: number;
}

// Factory option types
export interface FactoryOptions {
    saveToDb?: boolean;
    userId?: string;
}

export interface RoutineFactoryOptions extends FactoryOptions {
    mockResponses?: boolean;
}

export interface AgentFactoryOptions extends FactoryOptions {
    mockBehaviors?: boolean;
    teamId?: string;
}

export interface SwarmFactoryOptions extends FactoryOptions {
    mockState?: boolean;
}

// Event types
export interface TestEvent {
    topic: string;
    data: Record<string, unknown>;
    timestamp: Date;
    source: string;
}

export interface EventSubscription {
    topic: string;
    handler: (event: TestEvent) => void;
    cleanup?: () => void;
}

// Test execution types
export interface TestExecution {
    id: string;
    startTime: Date;
    endTime?: Date;
    events: TestEvent[];
    routineCalls: RoutineCallRecord[];
    blackboardState: Map<string, unknown>;
    resourceUsage: ResourceUsage;
}

export interface RoutineCallRecord {
    id: string;
    routineId: string;
    routineLabel: string;
    input: Record<string, unknown>;
    output: Record<string, unknown>;
    timestamp: Date;
    duration: number;
    success: boolean;
    error?: Error;
}

export interface ResourceUsage {
    creditsUsed: string;
    durationMs: number;
    memoryUsedMB: number;
    stepsExecuted: number;
    startTime: Date;
}

// Validation types
export interface ValidationResult {
    valid: boolean;
    errors: ValidationError[];
    warnings: ValidationWarning[];
}

export interface ValidationError {
    type: string;
    message: string;
    field?: string;
    expected?: unknown;
    actual?: unknown;
}

export interface ValidationWarning {
    type: string;
    message: string;
    field?: string;
}

// Assertion types
export interface AssertionContext {
    testName: string;
    scenario: string;
    timestamp: Date;
}

// Utility types
export type DeepPartial<T> = {
    [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

export type TestDataBuilder<T> = {
    [K in keyof T]: T[K] extends (...args: any[]) => any 
        ? T[K] 
        : () => T[K];
};

// Error types
export class ExecutionTestError extends Error {
    constructor(
        message: string,
        public readonly code: string,
        public readonly context?: Record<string, unknown>,
    ) {
        super(message);
        this.name = "ExecutionTestError";
    }
}

export class SchemaValidationError extends ExecutionTestError {
    constructor(message: string, public readonly schemaPath: string) {
        super(message, "SCHEMA_VALIDATION_ERROR", { schemaPath });
    }
}

export class MockConfigurationError extends ExecutionTestError {
    constructor(message: string, public readonly mockType: string) {
        super(message, "MOCK_CONFIGURATION_ERROR", { mockType });
    }
}

export class ScenarioExecutionError extends ExecutionTestError {
    constructor(message: string, public readonly scenarioName: string) {
        super(message, "SCENARIO_EXECUTION_ERROR", { scenarioName });
    }
}
