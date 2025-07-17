/**
 * Scenario Factory Types
 * 
 * Type definitions for scenario testing framework
 */

export interface ScenarioDefinition {
    name: string;
    description: string;
    schemas: {
        routines: string[];    // Paths to routine schemas
        agents: string[];      // Paths to agent schemas  
        swarms: string[];      // Paths to swarm schemas
    };
    mockConfig: MockConfiguration;
    expectations: ScenarioExpectations;
}

export interface MockConfiguration {
    ai?: Record<string, MockAIConfig>;
    routines?: Record<string, MockRoutineConfig>;
    timing?: Record<string, number>;
    infrastructure?: {
        redis?: MockRedisConfig;
        apps?: MockAppConfig[];
    };
}

export interface MockAIConfig {
    responses: Array<{
        attempt?: number;
        input?: any;
        response: any;
        delay?: number;
    }>;
}

export interface MockRoutineConfig {
    responses: Array<{
        attempt?: number;
        success: boolean;
        output?: any;
        logs?: string;
        error?: string;
    }>;
}

export interface MockRedisConfig {
    connectionFailures?: number;
    latency?: number;
}

export interface MockAppConfig {
    name: string;
    startDelay?: number;
    startFailures?: number;
    healthCheckResponses?: boolean[];
}

export interface ScenarioExpectations {
    initialBlackboard?: Record<string, any>;
    finalBlackboard?: Record<string, any>;
    eventSequence?: string[];
    routineCalls?: Array<{
        routine: string;
        times: number;
    }>;
    duration?: {
        min?: number;
        max?: number;
    };
    outcomes?: string[];
}

export interface ScenarioContext {
    name: string;
    description: string;
    routines: any[];
    agents: any[];
    swarms: any[];
    blackboard: Map<string, any>;
    mockController: any;
    expectations: ScenarioExpectations;
    events: ScenarioEvent[];
    routineCalls: RoutineCall[];
    startTime: Date;
    endTime?: Date;
    // Execution services
    eventInterceptor?: any;
    contextManager?: any;
    stepExecutor?: any;
}

export interface ScenarioEvent {
    topic: string;
    data: any;
    timestamp: Date;
    source: string;
}

export interface RoutineCall {
    routineId: string;
    routineLabel: string;
    input: any;
    output: any;
    timestamp: Date;
    duration: number;
    success: boolean;
}

export interface ScenarioResult {
    success: boolean;
    blackboard: Record<string, unknown>;
    events: ScenarioEvent[];
    routineCalls: RoutineCall[];
    attempts?: number;
    finalStatus?: string;
    duration: number;
    resourceUsage?: import("./ResourceManager.js").ResourceUsage;
    errors?: Error[];
}
