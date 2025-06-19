import { generatePublicId } from "@vrooli/shared";
import {
    type SwarmAgent,
    type AgentRole,
    type AgentNorm,
    type AgentPerformance,
    type SwarmTeam,
    type TeamHierarchy,
    type TeamFormation,
    type TeamConstraints,
    type AgentCapability,
} from "@vrooli/shared";
import { TEST_IDS, TestIdFactory } from "../../testIdGenerator.js";

/**
 * Database fixtures for Agent/Team models - used for seeding test data
 * These support the MOISE+ organizational model for multi-agent systems
 */

// Consistent IDs for testing - using pre-generated IDs
export const agentDbIds = {
    agent1: TEST_IDS.AGENT_1,
    agent2: TEST_IDS.AGENT_2,
    agent3: TEST_IDS.AGENT_3,
    agent4: TEST_IDS.AGENT_4,
    team1: TestIdFactory.team(1),
    team2: TestIdFactory.team(2),
    team3: TestIdFactory.team(3),
    role1: TestIdFactory.team(101), // Using team factory for role IDs
    role2: TestIdFactory.team(102),
    role3: TestIdFactory.team(103),
    role4: TestIdFactory.team(104),
    role5: TestIdFactory.team(105),
};

/**
 * Predefined agent roles following MOISE+ structure
 */
export const agentRoles = {
    coordinator: {
        id: agentDbIds.role1,
        name: "Coordinator",
        description: "Manages team coordination and goal decomposition",
        capabilities: ["planning", "resource_management", "communication", "negotiation"],
        responsibilities: [
            "Goal decomposition",
            "Resource allocation",
            "Conflict resolution",
            "Progress monitoring",
        ],
        norms: [
            { type: "obligation", target: "report_progress" },
            { type: "obligation", target: "allocate_resources_fairly" },
            { type: "permission", target: "modify_goals" },
            { type: "permission", target: "reassign_tasks" },
        ],
    } as AgentRole,
    
    analyzer: {
        id: agentDbIds.role2,
        name: "Analyzer",
        description: "Performs data analysis and reasoning tasks",
        capabilities: ["reasoning", "learning", "monitoring"],
        responsibilities: [
            "Data analysis",
            "Pattern recognition",
            "Insight generation",
            "Performance analysis",
        ],
        norms: [
            { type: "obligation", target: "validate_results" },
            { type: "permission", target: "access_data" },
            { type: "prohibition", target: "modify_raw_data" },
        ],
    } as AgentRole,
    
    executor: {
        id: agentDbIds.role3,
        name: "Executor",
        description: "Executes assigned tasks and operations",
        capabilities: ["execution", "monitoring", "communication"],
        responsibilities: [
            "Task execution",
            "Status reporting",
            "Error handling",
            "Resource usage tracking",
        ],
        norms: [
            { type: "obligation", target: "complete_assigned_tasks" },
            { type: "obligation", target: "report_errors" },
            { type: "prohibition", target: "exceed_resource_limits" },
        ],
    } as AgentRole,
    
    monitor: {
        id: agentDbIds.role4,
        name: "Monitor",
        description: "Monitors system health and performance",
        capabilities: ["monitoring", "communication"],
        responsibilities: [
            "System monitoring",
            "Performance tracking",
            "Anomaly detection",
            "Alert generation",
        ],
        norms: [
            { type: "obligation", target: "continuous_monitoring" },
            { type: "obligation", target: "alert_on_anomalies" },
            { type: "permission", target: "access_all_metrics" },
        ],
    } as AgentRole,
    
    learner: {
        id: agentDbIds.role5,
        name: "Learner",
        description: "Learns from experience and improves strategies",
        capabilities: ["learning", "reasoning", "planning"],
        responsibilities: [
            "Pattern learning",
            "Strategy optimization",
            "Knowledge sharing",
            "Adaptation recommendations",
        ],
        norms: [
            { type: "obligation", target: "share_learnings" },
            { type: "permission", target: "experiment_safely" },
            { type: "permission", target: "propose_improvements" },
        ],
    } as AgentRole,
};

/**
 * Default agent performance metrics
 */
export const defaultPerformance: AgentPerformance = {
    tasksCompleted: 0,
    tasksFailled: 0,
    averageCompletionTime: 0,
    successRate: 1,
    resourceEfficiency: 1,
    collaborationScore: 1,
    averageExecutionTime: 0,
};

/**
 * Minimal agent
 */
export const minimalAgent: SwarmAgent = {
    id: agentDbIds.agent1,
    name: "Basic Agent",
    role: agentRoles.executor,
    state: "idle",
    capabilities: agentRoles.executor.capabilities,
    performance: defaultPerformance,
};

/**
 * Active agent with current task
 */
export const activeAgent: SwarmAgent = {
    id: agentDbIds.agent2,
    name: "Active Executor",
    role: agentRoles.executor,
    state: "active",
    capabilities: agentRoles.executor.capabilities,
    currentTask: "process_data_batch_001",
    performance: {
        tasksCompleted: 15,
        tasksFailled: 1,
        averageCompletionTime: 45000,
        successRate: 0.94,
        resourceEfficiency: 0.87,
        collaborationScore: 0.9,
        averageExecutionTime: 42000,
    },
};

/**
 * High-performing coordinator agent
 */
export const coordinatorAgent: SwarmAgent = {
    id: agentDbIds.agent3,
    name: "Lead Coordinator",
    role: agentRoles.coordinator,
    state: "active",
    capabilities: [...agentRoles.coordinator.capabilities, "learning"],
    currentTask: "coordinate_team_alpha",
    performance: {
        tasksCompleted: 50,
        tasksFailled: 2,
        averageCompletionTime: 60000,
        successRate: 0.96,
        resourceEfficiency: 0.92,
        collaborationScore: 0.95,
        averageExecutionTime: 58000,
    },
};

/**
 * Minimal team
 */
export const minimalTeam: SwarmTeam = {
    id: agentDbIds.team1,
    name: "Basic Team",
    goal: "Complete assigned tasks",
    agents: [minimalAgent],
    hierarchy: {
        structure: "flat",
        relationships: [],
    },
    formation: new Date(),
    status: "active",
};

/**
 * Hierarchical team
 */
export const hierarchicalTeam: SwarmTeam = {
    id: agentDbIds.team2,
    name: "Hierarchical Team",
    goal: "Complex data processing with coordination",
    agents: [
        coordinatorAgent,
        activeAgent,
        {
            ...minimalAgent,
            id: agentDbIds.agent4,
            name: "Worker Agent",
            state: "active",
        },
    ],
    hierarchy: {
        leader: coordinatorAgent.id,
        structure: "hierarchical",
        relationships: [
            {
                from: activeAgent.id,
                to: coordinatorAgent.id,
                type: "reports_to",
            },
            {
                from: agentDbIds.agent4,
                to: coordinatorAgent.id,
                type: "reports_to",
            },
        ],
    },
    formation: new Date(),
    status: "active",
};

/**
 * Matrix organization team
 */
export const matrixTeam: SwarmTeam = {
    id: agentDbIds.team3,
    name: "Matrix Team",
    goal: "Collaborative problem solving",
    agents: [
        coordinatorAgent,
        {
            ...activeAgent,
            role: agentRoles.analyzer,
            capabilities: agentRoles.analyzer.capabilities,
        },
        {
            ...minimalAgent,
            id: generatePK(),
            role: agentRoles.monitor,
            capabilities: agentRoles.monitor.capabilities,
        },
        {
            id: generatePK(),
            name: "Learning Agent",
            role: agentRoles.learner,
            state: "active",
            capabilities: agentRoles.learner.capabilities,
            performance: defaultPerformance,
        },
    ],
    hierarchy: {
        leader: coordinatorAgent.id,
        structure: "matrix",
        relationships: [
            // Supervision relationships
            {
                from: coordinatorAgent.id,
                to: activeAgent.id,
                type: "supervises",
            },
            // Collaboration relationships
            {
                from: activeAgent.id,
                to: agentDbIds.agent3,
                type: "collaborates_with",
            },
        ],
    },
    formation: new Date(),
    status: "active",
};

/**
 * Factory for creating agent fixtures with overrides
 */
export class AgentFactory {
    /**
     * Create minimal agent
     */
    static createMinimal(
        role?: AgentRole,
        overrides?: Partial<SwarmAgent>
    ): SwarmAgent {
        return {
            ...minimalAgent,
            id: generatePK(),
            role: role || agentRoles.executor,
            capabilities: role?.capabilities || agentRoles.executor.capabilities,
            ...overrides,
        };
    }

    /**
     * Create agent with specific role
     */
    static createWithRole(
        roleName: keyof typeof agentRoles,
        overrides?: Partial<SwarmAgent>
    ): SwarmAgent {
        const role = agentRoles[roleName];
        return {
            id: generatePK(),
            name: `${role.name} Agent`,
            role,
            state: "idle",
            capabilities: role.capabilities,
            performance: defaultPerformance,
            ...overrides,
        };
    }

    /**
     * Create active agent with performance history
     */
    static createActive(
        tasksCompleted: number = 10,
        overrides?: Partial<SwarmAgent>
    ): SwarmAgent {
        const tasksFailled = Math.floor(tasksCompleted * 0.05); // 5% failure rate
        const successRate = tasksCompleted / (tasksCompleted + tasksFailled);
        
        return {
            ...activeAgent,
            id: generatePK(),
            performance: {
                tasksCompleted,
                tasksFailled,
                averageCompletionTime: 30000 + Math.random() * 30000,
                successRate,
                resourceEfficiency: 0.7 + Math.random() * 0.3,
                collaborationScore: 0.8 + Math.random() * 0.2,
                averageExecutionTime: 25000 + Math.random() * 25000,
            },
            ...overrides,
        };
    }

    /**
     * Create agent with specific capabilities
     */
    static createWithCapabilities(
        capabilities: AgentCapability[],
        overrides?: Partial<SwarmAgent>
    ): SwarmAgent {
        // Find best matching role
        let bestRole = agentRoles.executor;
        let maxMatch = 0;
        
        for (const [_, role] of Object.entries(agentRoles)) {
            const match = capabilities.filter(cap => 
                role.capabilities.includes(cap)
            ).length;
            if (match > maxMatch) {
                maxMatch = match;
                bestRole = role;
            }
        }

        return this.createMinimal(bestRole, {
            capabilities,
            ...overrides,
        });
    }

    /**
     * Create learning agent
     */
    static createLearningAgent(overrides?: Partial<SwarmAgent>): SwarmAgent {
        return this.createWithRole("learner", {
            state: "active",
            performance: {
                ...defaultPerformance,
                tasksCompleted: 25,
                successRate: 0.98,
                resourceEfficiency: 0.95,
            },
            ...overrides,
        });
    }
}

/**
 * Factory for creating team fixtures with overrides
 */
export class TeamFactory {
    /**
     * Create minimal team
     */
    static createMinimal(overrides?: Partial<SwarmTeam>): SwarmTeam {
        return {
            ...minimalTeam,
            id: generatePK(),
            formation: new Date(),
            agents: [AgentFactory.createMinimal()],
            ...overrides,
        };
    }

    /**
     * Create team with specific structure
     */
    static createWithStructure(
        structure: "flat" | "hierarchical" | "matrix",
        agentCount: number = 3,
        overrides?: Partial<SwarmTeam>
    ): SwarmTeam {
        const agents: SwarmAgent[] = [];
        const relationships: TeamHierarchy["relationships"] = [];
        
        // Always add a coordinator for non-flat structures
        const coordinatorId = generatePK();
        if (structure !== "flat") {
            agents.push(AgentFactory.createWithRole("coordinator", {
                id: coordinatorId,
                state: "active",
            }));
        }

        // Add other agents
        const remainingCount = structure === "flat" ? agentCount : agentCount - 1;
        for (let i = 0; i < remainingCount; i++) {
            const agent = AgentFactory.createWithRole(
                i % 2 === 0 ? "executor" : "analyzer",
                { state: i === 0 ? "active" : "idle" }
            );
            agents.push(agent);

            // Add relationships
            if (structure === "hierarchical" && agents.length > 1) {
                relationships.push({
                    from: agent.id,
                    to: coordinatorId,
                    type: "reports_to",
                });
            } else if (structure === "matrix") {
                if (agents.length > 1) {
                    relationships.push({
                        from: coordinatorId,
                        to: agent.id,
                        type: "supervises",
                    });
                }
                // Add collaboration between non-coordinator agents
                if (i > 0 && structure === "matrix") {
                    relationships.push({
                        from: agents[agents.length - 2].id,
                        to: agent.id,
                        type: "collaborates_with",
                    });
                }
            }
        }

        return {
            id: generatePK(),
            name: `${structure} Team`,
            goal: "Achieve team objectives",
            agents,
            hierarchy: {
                leader: structure !== "flat" ? coordinatorId : undefined,
                structure,
                relationships,
            },
            formation: new Date(),
            status: "active",
            ...overrides,
        };
    }

    /**
     * Create specialized team
     */
    static createSpecialized(
        specialization: "analysis" | "execution" | "monitoring" | "learning",
        overrides?: Partial<SwarmTeam>
    ): SwarmTeam {
        const roleMap = {
            analysis: ["coordinator", "analyzer", "analyzer"],
            execution: ["coordinator", "executor", "executor", "monitor"],
            monitoring: ["monitor", "monitor", "analyzer"],
            learning: ["learner", "analyzer", "coordinator"],
        };

        const roles = roleMap[specialization];
        const agents = roles.map((roleName, index) => 
            AgentFactory.createWithRole(
                roleName as keyof typeof agentRoles,
                {
                    state: index === 0 ? "active" : "idle",
                    name: `${roleName} ${index + 1}`,
                }
            )
        );

        return {
            id: generatePK(),
            name: `${specialization} Specialist Team`,
            goal: `Specialized ${specialization} operations`,
            agents,
            hierarchy: {
                leader: agents[0].id,
                structure: agents.length > 3 ? "hierarchical" : "flat",
                relationships: agents.slice(1).map(agent => ({
                    from: agent.id,
                    to: agents[0].id,
                    type: "reports_to" as const,
                })),
            },
            formation: new Date(),
            status: "active",
            ...overrides,
        };
    }

    /**
     * Create team formation with constraints
     */
    static createFormation(
        constraints?: TeamConstraints,
        overrides?: Partial<TeamFormation>
    ): TeamFormation {
        const requiredCaps = constraints?.requiredCapabilities || ["execution"];
        const minSize = constraints?.minSize || 1;
        const maxSize = constraints?.maxSize || 5;
        const size = minSize + Math.floor(Math.random() * (maxSize - minSize + 1));
        
        const agents: SwarmAgent[] = [];
        
        // Ensure we have agents with required capabilities
        for (const cap of requiredCaps) {
            if (agents.length < size) {
                agents.push(
                    AgentFactory.createWithCapabilities([cap], {
                        state: agents.length === 0 ? "active" : "idle",
                    })
                );
            }
        }

        // Fill remaining slots
        while (agents.length < size) {
            agents.push(AgentFactory.createMinimal());
        }

        return {
            id: generatePK(),
            swarmId: generatePK(),
            agents,
            purpose: "Meet specified constraints",
            constraints,
            createdAt: new Date(),
            ...overrides,
        };
    }
}

/**
 * Helper to create custom role
 */
export function createCustomRole(
    name: string,
    capabilities: AgentCapability[],
    overrides?: Partial<AgentRole>
): AgentRole {
    return {
        id: generatePK(),
        name,
        description: `Custom role: ${name}`,
        capabilities,
        responsibilities: [`Perform ${name} duties`],
        norms: [
            { type: "obligation", target: "follow_protocols" },
        ],
        ...overrides,
    };
}

/**
 * Helper to calculate team performance
 */
export function calculateTeamPerformance(team: SwarmTeam): AgentPerformance {
    const performances = team.agents.map(a => a.performance);
    const count = performances.length;
    
    if (count === 0) return defaultPerformance;

    return {
        tasksCompleted: performances.reduce((sum, p) => sum + p.tasksCompleted, 0),
        tasksFailled: performances.reduce((sum, p) => sum + p.tasksFailled, 0),
        averageCompletionTime: performances.reduce((sum, p) => 
            sum + p.averageCompletionTime, 0) / count,
        successRate: performances.reduce((sum, p) => sum + p.successRate, 0) / count,
        resourceEfficiency: performances.reduce((sum, p) => 
            sum + p.resourceEfficiency, 0) / count,
        collaborationScore: performances.reduce((sum, p) => 
            sum + (p.collaborationScore || 0), 0) / count,
        averageExecutionTime: performances.reduce((sum, p) => 
            sum + (p.averageExecutionTime || 0), 0) / count,
    };
}

/**
 * Helper to seed test agents and teams
 */
export async function seedTestAgentsAndTeams(
    prisma: any,
    options?: {
        agentCount?: number;
        teamCount?: number;
        includeSpecialized?: boolean;
    }
) {
    const agents = [];
    const teams = [];
    
    // Create individual agents
    const agentCount = options?.agentCount || 5;
    for (let i = 0; i < agentCount; i++) {
        const roleNames = Object.keys(agentRoles);
        const roleName = roleNames[i % roleNames.length] as keyof typeof agentRoles;
        agents.push(AgentFactory.createWithRole(roleName, {
            name: `Test Agent ${i + 1}`,
        }));
    }

    // Create teams
    const teamCount = options?.teamCount || 3;
    const structures: Array<"flat" | "hierarchical" | "matrix"> = 
        ["flat", "hierarchical", "matrix"];
    
    for (let i = 0; i < teamCount; i++) {
        teams.push(TeamFactory.createWithStructure(
            structures[i % structures.length],
            3,
            { name: `Test Team ${i + 1}` }
        ));
    }

    // Add specialized teams if requested
    if (options?.includeSpecialized) {
        teams.push(
            TeamFactory.createSpecialized("analysis", { name: "Analysis Team" }),
            TeamFactory.createSpecialized("execution", { name: "Execution Team" }),
            TeamFactory.createSpecialized("learning", { name: "Learning Team" })
        );
    }

    return { agents, teams };
}