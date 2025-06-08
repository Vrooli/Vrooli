import { generatePublicId } from "@vrooli/shared";
import {
    type Swarm,
    type SwarmConfig,
    type SwarmState,
    type SwarmAgent,
    type SwarmTeam,
    type TeamFormation,
    type AgentRole,
    type ChatConfigObject,
    type SwarmResource,
    type SwarmSubTask,
    type BlackboardItem,
    type ResourceLimits,
    type SchedulingConfig,
    SwarmState as SwarmStateEnum,
} from "@vrooli/shared";
import { TEST_IDS, TestIdFactory } from "./testIdGenerator.js";

/**
 * Database fixtures for Swarm model - used for seeding test data
 * These support the three-tier AI execution architecture
 */

// Consistent IDs for testing
export const swarmDbIds = {
    swarm1: TEST_IDS.SWARM_1,
    swarm2: TEST_IDS.SWARM_2,
    swarm3: TEST_IDS.SWARM_3,
    agent1: TEST_IDS.AGENT_1,
    agent2: TEST_IDS.AGENT_2,
    agent3: TEST_IDS.AGENT_3,
    team1: TestIdFactory.team(1),
    team2: TestIdFactory.team(2),
    role1: TestIdFactory.team(101),
    role2: TestIdFactory.team(102),
    resource1: TestIdFactory.routine(1001),
    resource2: TestIdFactory.routine(1002),
    subtask1: TestIdFactory.context(2001),
    subtask2: TestIdFactory.context(2002),
    blackboard1: TestIdFactory.context(3001),
};

/**
 * Default agent roles
 */
export const defaultRoles = {
    coordinator: {
        id: swarmDbIds.role1,
        name: "Coordinator",
        description: "Manages swarm coordination and goal decomposition",
        capabilities: ["planning", "resource_management", "communication"],
        responsibilities: ["goal decomposition", "resource allocation", "conflict resolution"],
        norms: [
            { type: "obligation", target: "report_progress" },
            { type: "permission", target: "allocate_resources" },
        ],
    } as AgentRole,
    executor: {
        id: swarmDbIds.role2,
        name: "Executor",
        description: "Executes assigned tasks and reports results",
        capabilities: ["execution", "monitoring", "communication"],
        responsibilities: ["task execution", "status reporting", "error handling"],
        norms: [
            { type: "obligation", target: "complete_tasks" },
            { type: "prohibition", target: "modify_goals" },
        ],
    } as AgentRole,
};

/**
 * Minimal swarm configuration
 */
export const minimalSwarmConfig: SwarmConfig = {
    maxAgents: 10,
    minAgents: 1,
    consensusThreshold: 0.7,
    decisionTimeout: 30000,
    adaptationInterval: 60000,
    resourceOptimization: true,
    learningEnabled: false,
};

/**
 * Minimal chat config
 */
export const minimalChatConfig: ChatConfigObject = {
    goal: "Test goal",
    subtasks: [],
    resources: [],
    blackboard: [],
    limits: {
        maxTokens: 1000,
        maxTime: 300000,
        maxCost: 10,
    },
    scheduling: {
        mode: "SEQUENTIAL",
    },
    pendingToolCalls: [],
    createdAt: new Date(),
    updatedAt: new Date(),
};

/**
 * Minimal swarm data
 */
export const minimalSwarm: Partial<Swarm> = {
    id: swarmDbIds.swarm1,
    name: "Test Swarm",
    description: "A minimal test swarm",
    state: SwarmStateEnum.UNINITIALIZED,
    config: minimalSwarmConfig,
    metadata: {},
    createdAt: new Date(),
};

/**
 * Swarm with team formation
 */
export const swarmWithTeam: Partial<Swarm> = {
    id: swarmDbIds.swarm2,
    name: "Team Swarm",
    description: "A swarm with team formation",
    state: SwarmStateEnum.FORMING,
    config: minimalSwarmConfig,
    team: {
        id: swarmDbIds.team1,
        swarmId: swarmDbIds.swarm2,
        agents: [
            {
                id: swarmDbIds.agent1,
                name: "Coordinator Agent",
                role: defaultRoles.coordinator,
                state: "active",
                capabilities: defaultRoles.coordinator.capabilities,
                performance: {
                    tasksCompleted: 0,
                    tasksFailled: 0,
                    averageCompletionTime: 0,
                    successRate: 1,
                    resourceEfficiency: 1,
                },
            },
            {
                id: swarmDbIds.agent2,
                name: "Executor Agent",
                role: defaultRoles.executor,
                state: "active",
                capabilities: defaultRoles.executor.capabilities,
                performance: {
                    tasksCompleted: 0,
                    tasksFailled: 0,
                    averageCompletionTime: 0,
                    successRate: 1,
                    resourceEfficiency: 1,
                },
            },
        ],
        createdAt: new Date(),
    } as TeamFormation,
    metadata: {
        conversationId: generatePublicId(),
        initiatingUserId: generatePublicId(),
    },
    createdAt: new Date(),
};

/**
 * Complete swarm with all features
 */
export const completeSwarm: Partial<Swarm> = {
    id: swarmDbIds.swarm3,
    name: "Complete Swarm",
    description: "A fully configured swarm with all features",
    state: SwarmStateEnum.EXECUTING,
    config: {
        ...minimalSwarmConfig,
        maxBudget: 100,
        maxDuration: 3600000,
        learningEnabled: true,
    },
    team: {
        id: swarmDbIds.team2,
        swarmId: swarmDbIds.swarm3,
        agents: [
            {
                id: swarmDbIds.agent1,
                name: "Lead Coordinator",
                role: defaultRoles.coordinator,
                state: "active",
                capabilities: [...defaultRoles.coordinator.capabilities, "learning", "negotiation"],
                currentTask: "goal_decomposition",
                performance: {
                    tasksCompleted: 10,
                    tasksFailled: 1,
                    averageCompletionTime: 45000,
                    successRate: 0.91,
                    resourceEfficiency: 0.85,
                    collaborationScore: 0.9,
                },
            },
            {
                id: swarmDbIds.agent2,
                name: "Primary Executor",
                role: defaultRoles.executor,
                state: "active",
                capabilities: defaultRoles.executor.capabilities,
                currentTask: "data_processing",
                performance: {
                    tasksCompleted: 25,
                    tasksFailled: 2,
                    averageCompletionTime: 30000,
                    successRate: 0.93,
                    resourceEfficiency: 0.88,
                    collaborationScore: 0.85,
                },
            },
            {
                id: swarmDbIds.agent3,
                name: "Secondary Executor",
                role: defaultRoles.executor,
                state: "idle",
                capabilities: [...defaultRoles.executor.capabilities, "reasoning"],
                performance: {
                    tasksCompleted: 15,
                    tasksFailled: 1,
                    averageCompletionTime: 35000,
                    successRate: 0.94,
                    resourceEfficiency: 0.9,
                    collaborationScore: 0.87,
                },
            },
        ],
        purpose: "Execute complex data analysis workflow",
        constraints: {
            requiredCapabilities: ["execution", "reasoning", "monitoring"],
            maxSize: 5,
            minSize: 2,
            budgetLimit: 100,
        },
        createdAt: new Date(),
    } as TeamFormation,
    metadata: {
        conversationId: generatePublicId(),
        initiatingUserId: generatePublicId(),
        projectId: generatePublicId(),
        priority: "high",
        tags: ["data-analysis", "automated"],
    },
    createdAt: new Date(),
    updatedAt: new Date(),
};

/**
 * Factory for creating swarm fixtures with overrides
 */
export class SwarmFactory {
    static createMinimal(overrides?: Partial<Swarm>): Partial<Swarm> {
        return {
            ...minimalSwarm,
            id: generatePK(),
            createdAt: new Date(),
            ...overrides,
        };
    }

    static createWithTeam(
        agentCount: number = 2,
        overrides?: Partial<Swarm>
    ): Partial<Swarm> {
        const agents: SwarmAgent[] = [];
        
        // Always add a coordinator
        agents.push({
            id: generatePK(),
            name: "Coordinator Agent",
            role: defaultRoles.coordinator,
            state: "active",
            capabilities: defaultRoles.coordinator.capabilities,
            performance: {
                tasksCompleted: 0,
                tasksFailled: 0,
                averageCompletionTime: 0,
                successRate: 1,
                resourceEfficiency: 1,
            },
        });

        // Add executors
        for (let i = 1; i < agentCount; i++) {
            agents.push({
                id: generatePK(),
                name: `Executor Agent ${i}`,
                role: defaultRoles.executor,
                state: "active",
                capabilities: defaultRoles.executor.capabilities,
                performance: {
                    tasksCompleted: 0,
                    tasksFailled: 0,
                    averageCompletionTime: 0,
                    successRate: 1,
                    resourceEfficiency: 1,
                },
            });
        }

        const swarmId = generatePK();
        return {
            ...swarmWithTeam,
            id: swarmId,
            team: {
                id: generatePK(),
                swarmId,
                agents,
                createdAt: new Date(),
            },
            createdAt: new Date(),
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Swarm>): Partial<Swarm> {
        return {
            ...completeSwarm,
            id: generatePK(),
            createdAt: new Date(),
            updatedAt: new Date(),
            ...overrides,
        };
    }

    /**
     * Create a swarm in a specific state
     */
    static createInState(
        state: SwarmState,
        overrides?: Partial<Swarm>
    ): Partial<Swarm> {
        const baseSwarm = state === SwarmStateEnum.UNINITIALIZED
            ? this.createMinimal()
            : this.createWithTeam();

        return {
            ...baseSwarm,
            state,
            ...overrides,
        };
    }

    /**
     * Create a swarm with chat configuration
     */
    static createWithChatConfig(
        chatConfig: Partial<ChatConfigObject>,
        overrides?: Partial<Swarm>
    ): Partial<Swarm> {
        const swarm = this.createWithTeam(2, overrides);
        
        return {
            ...swarm,
            metadata: {
                ...swarm.metadata,
                chatConfig: {
                    ...minimalChatConfig,
                    ...chatConfig,
                },
            },
        };
    }

    /**
     * Create a swarm with resources and subtasks
     */
    static createWithGoalDecomposition(
        overrides?: Partial<Swarm>
    ): Partial<Swarm> {
        const resources: SwarmResource[] = [
            {
                id: generatePK(),
                type: "Routine",
                name: "Data Processing Routine",
                description: "Process and analyze input data",
                usedFor: "Context",
            },
            {
                id: generatePK(),
                type: "Api",
                name: "Analysis API",
                description: "External analysis service",
                usedFor: "Improvement",
            },
        ];

        const subtasks: SwarmSubTask[] = [
            {
                description: "Load and validate input data",
                status: "completed",
                result: "Data loaded successfully",
            },
            {
                description: "Process data through analysis pipeline",
                status: "inProgress",
            },
            {
                description: "Generate report from results",
                status: "pending",
            },
        ];

        const blackboard: BlackboardItem[] = [
            {
                id: generatePK(),
                value: "data_source: s3://bucket/data.csv",
                createdAt: new Date(),
                updatedAt: new Date(),
                createdBy: swarmDbIds.agent1,
            },
            {
                id: generatePK(),
                value: "processing_status: 45% complete",
                createdAt: new Date(),
                updatedAt: new Date(),
                createdBy: swarmDbIds.agent2,
            },
        ];

        return this.createWithChatConfig(
            {
                goal: "Analyze customer data and generate insights report",
                subtasks,
                resources,
                blackboard,
                limits: {
                    maxTokens: 5000,
                    maxTime: 600000,
                    maxCost: 50,
                    maxApiCalls: 100,
                },
                scheduling: {
                    mode: "PARALLEL",
                    maxConcurrency: 3,
                    timeout: 300000,
                },
            },
            overrides
        );
    }
}

/**
 * Helper to create swarm team structure
 */
export function createSwarmTeam(
    swarmId: string,
    options?: {
        agentCount?: number;
        hierarchy?: "flat" | "hierarchical" | "matrix";
        includeBot?: boolean;
    }
): SwarmTeam {
    const agentCount = options?.agentCount || 3;
    const hierarchy = options?.hierarchy || "flat";
    const agents: SwarmAgent[] = [];

    // Add coordinator
    const coordinatorId = generatePK();
    agents.push({
        id: coordinatorId,
        name: "Team Coordinator",
        role: defaultRoles.coordinator,
        state: "active",
        capabilities: defaultRoles.coordinator.capabilities,
        performance: {
            tasksCompleted: 5,
            tasksFailled: 0,
            averageCompletionTime: 60000,
            successRate: 1,
            resourceEfficiency: 0.9,
        },
    });

    // Add executors
    for (let i = 1; i < agentCount; i++) {
        agents.push({
            id: generatePK(),
            name: `Team Member ${i}`,
            role: defaultRoles.executor,
            state: i === 1 ? "active" : "idle",
            capabilities: defaultRoles.executor.capabilities,
            performance: {
                tasksCompleted: Math.floor(Math.random() * 10),
                tasksFailled: Math.floor(Math.random() * 2),
                averageCompletionTime: 30000 + Math.random() * 30000,
                successRate: 0.8 + Math.random() * 0.2,
                resourceEfficiency: 0.7 + Math.random() * 0.3,
            },
        });
    }

    // Add bot if requested
    if (options?.includeBot) {
        agents.push({
            id: generatePK(),
            name: "AI Assistant Bot",
            role: {
                ...defaultRoles.executor,
                name: "AI Assistant",
                capabilities: ["reasoning", "learning", "execution"],
            },
            state: "active",
            capabilities: ["reasoning", "learning", "execution"],
            performance: {
                tasksCompleted: 50,
                tasksFailled: 2,
                averageCompletionTime: 15000,
                successRate: 0.96,
                resourceEfficiency: 0.95,
            },
        });
    }

    const relationships = [];
    if (hierarchy === "hierarchical") {
        // Everyone reports to coordinator
        for (let i = 1; i < agents.length; i++) {
            relationships.push({
                from: agents[i].id,
                to: coordinatorId,
                type: "reports_to" as const,
            });
        }
    } else if (hierarchy === "matrix") {
        // Coordinator supervises, others collaborate
        for (let i = 1; i < agents.length; i++) {
            relationships.push({
                from: coordinatorId,
                to: agents[i].id,
                type: "supervises" as const,
            });
            
            // Add collaboration between executors
            if (i < agents.length - 1) {
                relationships.push({
                    from: agents[i].id,
                    to: agents[i + 1].id,
                    type: "collaborates_with" as const,
                });
            }
        }
    }

    return {
        id: generatePK(),
        name: `Team ${swarmId}`,
        goal: "Execute swarm objectives efficiently",
        agents,
        hierarchy: {
            leader: coordinatorId,
            structure: hierarchy,
            relationships,
        },
        formation: new Date(),
        status: "active",
    };
}

/**
 * Helper to seed test swarms
 */
export async function seedTestSwarms(
    prisma: any,
    count: number = 3,
    options?: {
        withTeams?: boolean;
        states?: SwarmState[];
        userId?: string;
    }
) {
    const swarms = [];
    const states = options?.states || [
        SwarmStateEnum.UNINITIALIZED,
        SwarmStateEnum.FORMING,
        SwarmStateEnum.EXECUTING,
    ];

    for (let i = 0; i < count; i++) {
        const state = states[i % states.length];
        const swarmData = options?.withTeams
            ? SwarmFactory.createWithTeam(3, {
                name: `Test Swarm ${i + 1}`,
                state,
                metadata: {
                    conversationId: generatePublicId(),
                    initiatingUserId: options?.userId || generatePublicId(),
                },
            })
            : SwarmFactory.createMinimal({
                name: `Test Swarm ${i + 1}`,
                state,
            });

        // Note: You'll need to adapt this to your actual database schema
        // This is a conceptual example
        swarms.push(swarmData);
    }

    return swarms;
}