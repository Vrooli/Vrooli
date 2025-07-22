import { generatePK } from "@vrooli/shared";
import type { SwarmExecutionTask, RunTask } from "../../../tasks/taskTypes.js";
import { swarmTaskFactory } from "./swarmTaskFactory.js";
import { runTaskFactory } from "./runTaskFactory.js";

/**
 * Factory for creating multi-agent workflow test scenarios
 */
export interface MultiAgentScenario {
    name: string;
    description: string;
    agents: AgentFixture[];
    routines: RoutineFixture[];
    workflows: WorkflowStep[];
    expectedEvents: EventAssertion[];
    blackboardState: Record<string, unknown>;
}

export interface AgentFixture {
    id: string;
    name: string;
    role: "fixer" | "validator" | "coordinator" | "monitor";
    subscriptions: string[];
    behaviors: AgentBehavior[];
}

export interface AgentBehavior {
    trigger: { topic: string; when?: string };
    action: {
        type: "routine" | "invoke" | "emit";
        label?: string;
        eventType?: string;
        dataMapping?: Record<string, string>;
        outputOperations?: Record<string, unknown[]>;
    };
}

export interface RoutineFixture {
    id: string;
    name: string;
    type: "RoutineGenerate" | "RoutineMultiStep" | "RoutineData" | "RoutineAction";
    config: Record<string, unknown>;
    mockResponse?: unknown;
}

export interface WorkflowStep {
    stepNumber: number;
    expectedTrigger: string;
    expectedAction: string;
    expectedBlackboardUpdates?: Record<string, unknown>;
    expectedEvents?: string[];
}

export interface EventAssertion {
    eventType: string;
    expectedData: Record<string, unknown>;
    order: number;
}

/**
 * Redis Connection Fix Loop Scenario
 */
export function createRedisFixLoopScenario(): MultiAgentScenario {
    return {
        name: "redis-connection-fix-loop",
        description: "Multi-agent workflow for fixing and validating Redis connection issues",
        
        agents: [
            {
                id: generatePK().toString(),
                name: "redis-problem-fixer",
                role: "fixer",
                subscriptions: ["custom/redis/fix_requested"],
                behaviors: [
                    {
                        trigger: { topic: "custom/redis/fix_requested" },
                        action: {
                            type: "routine",
                            label: "redis-connection-fixer",
                            outputOperations: {
                                set: [
                                    { routineOutput: "fix_applied", blackboardId: "current_fix" },
                                    { routineOutput: "fix_description", blackboardId: "fix_details" },
                                ],
                            },
                        },
                    },
                    {
                        trigger: { topic: "custom/redis/fix_requested" },
                        action: {
                            type: "emit",
                            eventType: "custom/redis/ready_for_validation",
                            dataMapping: {
                                fixDetails: "blackboard.fix_details",
                                attemptNumber: "event.data.attemptNumber",
                            },
                        },
                    },
                ],
            },
            
            {
                id: generatePK().toString(),
                name: "redis-fix-validator",
                role: "validator",
                subscriptions: ["custom/redis/ready_for_validation"],
                behaviors: [
                    {
                        trigger: { topic: "custom/redis/ready_for_validation" },
                        action: {
                            type: "routine",
                            label: "redis-validation-workflow",
                            outputOperations: {
                                set: [
                                    { routineOutput: "validation_result", blackboardId: "validation_status" },
                                    { routineOutput: "log_analysis", blackboardId: "validation_logs" },
                                ],
                            },
                        },
                    },
                    {
                        trigger: { topic: "custom/redis/ready_for_validation" },
                        action: {
                            type: "emit",
                            eventType: "custom/redis/validation_complete",
                            dataMapping: {
                                success: "blackboard.validation_status.success",
                                attempt: "event.data.attemptNumber",
                                logs: "blackboard.validation_logs",
                            },
                        },
                    },
                ],
            },
            
            {
                id: generatePK().toString(),
                name: "redis-loop-coordinator",
                role: "coordinator",
                subscriptions: ["custom/redis/validation_complete"],
                behaviors: [
                    {
                        trigger: {
                            topic: "custom/redis/validation_complete",
                            when: "event.data.success == false && event.data.attempt < 5",
                        },
                        action: {
                            type: "routine",
                            label: "troubleshooting-doc-updater",
                        },
                    },
                    {
                        trigger: {
                            topic: "custom/redis/validation_complete",
                            when: "event.data.success == false && event.data.attempt < 5",
                        },
                        action: {
                            type: "emit",
                            eventType: "custom/redis/fix_requested",
                            dataMapping: {
                                attemptNumber: "event.data.attempt + 1",
                                previousLogs: "event.data.logs",
                            },
                        },
                    },
                ],
            },
        ],
        
        routines: [
            {
                id: generatePK().toString(),
                name: "redis-connection-fixer",
                type: "RoutineGenerate",
                config: {
                    callDataGenerate: {
                        schema: {
                            inputTemplate: {
                                issueDetails: "{{input.issue_details}}",
                                attemptNumber: "{{input.attempt_number}}",
                            },
                        },
                    },
                },
                mockResponse: {
                    fix_applied: "Updated connection pool settings",
                    fix_description: "Increased max connections and added retry logic",
                },
            },
            
            {
                id: generatePK().toString(),
                name: "redis-validation-workflow",
                type: "RoutineMultiStep",
                config: {
                    nodes: [
                        { id: "start_app", subroutineId: "app-starter" },
                        { id: "wait_period", subroutineId: "timer-wait-5min" },
                        { id: "collect_logs", subroutineId: "log-collector" },
                        { id: "analyze_logs", subroutineId: "redis-log-analyzer" },
                        { id: "stop_app", subroutineId: "app-stopper" },
                    ],
                    edges: [
                        { from: "start_app", to: "wait_period" },
                        { from: "wait_period", to: "collect_logs" },
                        { from: "collect_logs", to: "analyze_logs" },
                        { from: "analyze_logs", to: "stop_app" },
                    ],
                },
                mockResponse: {
                    validation_result: { success: false, errors: ["Connection timeout after 3 minutes"] },
                    log_analysis: { pattern: "redis_connection_error", frequency: 15 },
                },
            },
            
            {
                id: generatePK().toString(),
                name: "troubleshooting-doc-updater",
                type: "RoutineAction",
                config: {
                    callDataAction: {
                        schema: {
                            inputTemplate: {
                                attempt: "{{input.attempt}}",
                                logs: "{{input.logs}}",
                                fixTried: "{{input.fix_tried}}",
                            },
                        },
                    },
                },
            },
        ],
        
        workflows: [
            {
                stepNumber: 1,
                expectedTrigger: "custom/redis/fix_requested",
                expectedAction: "redis-connection-fixer routine execution",
                expectedBlackboardUpdates: {
                    current_fix: "Updated connection pool settings",
                    fix_details: "Increased max connections and added retry logic",
                },
                expectedEvents: ["custom/redis/ready_for_validation"],
            },
            {
                stepNumber: 2,
                expectedTrigger: "custom/redis/ready_for_validation",
                expectedAction: "redis-validation-workflow routine execution",
                expectedBlackboardUpdates: {
                    validation_status: { success: false },
                    validation_logs: { pattern: "redis_connection_error", frequency: 15 },
                },
                expectedEvents: ["custom/redis/validation_complete"],
            },
            {
                stepNumber: 3,
                expectedTrigger: "custom/redis/validation_complete",
                expectedAction: "troubleshooting-doc-updater routine execution",
                expectedEvents: ["custom/redis/fix_requested"],
            },
        ],
        
        expectedEvents: [
            {
                eventType: "custom/redis/fix_requested",
                expectedData: { attemptNumber: 1 },
                order: 1,
            },
            {
                eventType: "custom/redis/ready_for_validation",
                expectedData: { fixDetails: "Increased max connections and added retry logic" },
                order: 2,
            },
            {
                eventType: "custom/redis/validation_complete",
                expectedData: { success: false, attempt: 1 },
                order: 3,
            },
            {
                eventType: "custom/redis/fix_requested",
                expectedData: { attemptNumber: 2 },
                order: 4,
            },
        ],
        
        blackboardState: {
            redis_fix_workflow: {
                max_attempts: 5,
                current_attempt: 1,
                status: "in_progress",
            },
        },
    };
}

/**
 * Create SwarmExecutionTask from multi-agent scenario
 */
export function createSwarmTaskFromScenario(scenario: MultiAgentScenario): SwarmExecutionTask {
    return swarmTaskFactory({
        teamConfiguration: {
            leaderAgentId: scenario.agents[0].id,
            preferredTeamSize: scenario.agents.length,
            requiredSkills: scenario.agents.map(a => a.role),
            agents: scenario.agents.map(agent => ({
                id: agent.id,
                name: agent.name,
                role: agent.role,
                subscriptions: agent.subscriptions,
                behaviors: agent.behaviors,
            })),
        },
        scenario: scenario.name,
        blackboardInitialState: scenario.blackboardState,
    });
}

/**
 * Create multiple run tasks for scenario routines
 */
export function createRunTasksFromScenario(scenario: MultiAgentScenario): RunTask[] {
    return scenario.routines.map(routine => 
        runTaskFactory({
            routine: {
                id: routine.id,
                name: routine.name,
                type: routine.type,
                config: routine.config,
            },
            mockResponse: routine.mockResponse,
            scenario: scenario.name,
        }),
    );
}

/**
 * General multi-agent workflow factory
 */
export const multiAgentWorkflowFactory = {
    redisFixLoop: createRedisFixLoopScenario,
    createSwarmTask: createSwarmTaskFromScenario,
    createRunTasks: createRunTasksFromScenario,
};
