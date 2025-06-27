import { type EmergentAgent } from "../../../../services/execution/cross-cutting/agents/emergentAgent.js";
import { type EventPublisher } from "../../../../services/execution/shared/EventPublisher.js";

export interface BehaviorPattern {
    name: string;
    description: string;
    requiredCapabilities: string[];
    triggerConditions: TriggerCondition[];
    expectedBehaviors: ExpectedBehavior[];
    emergentOutcomes?: EmergentOutcome[];
}

export interface TriggerCondition {
    type: "event" | "metric" | "state" | "time";
    condition: any;
    description: string;
}

export interface ExpectedBehavior {
    action: string;
    probability: number;
    timing: "immediate" | "delayed" | "eventual";
    dependencies?: string[];
}

export interface EmergentOutcome {
    description: string;
    indicators: string[];
    minimumAgents: number;
}

export interface PatternTestResult {
    pattern: string;
    matched: boolean;
    behaviors: ObservedBehavior[];
    emergentOutcomes: string[];
    confidence: number;
}

export interface ObservedBehavior {
    agent: string;
    action: string;
    timestamp: Date;
    metrics: Record<string, any>;
}

/**
 * Common behavior patterns for emergent systems
 */
export const COMMON_PATTERNS = {
    SELF_ORGANIZATION: createPattern({
        name: "self-organization",
        description: "Agents organize without central control",
        requiredCapabilities: ["communicate", "coordinate"],
        triggerConditions: [
            {
                type: "metric",
                condition: { complexity: "> 0.7" },
                description: "High complexity triggers organization",
            },
        ],
        expectedBehaviors: [
            {
                action: "form_cluster",
                probability: 0.8,
                timing: "eventual",
            },
            {
                action: "establish_roles",
                probability: 0.6,
                timing: "delayed",
            },
        ],
        emergentOutcomes: [
            {
                description: "Hierarchical structure emerges",
                indicators: ["role_differentiation", "communication_patterns"],
                minimumAgents: 3,
            },
        ],
    }),

    COLLECTIVE_INTELLIGENCE: createPattern({
        name: "collective-intelligence",
        description: "Group solves problems beyond individual capability",
        requiredCapabilities: ["share_knowledge", "learn"],
        triggerConditions: [
            {
                type: "event",
                condition: { event: "complex_problem" },
                description: "Problem exceeds individual capacity",
            },
        ],
        expectedBehaviors: [
            {
                action: "share_partial_solution",
                probability: 0.9,
                timing: "immediate",
            },
            {
                action: "combine_solutions",
                probability: 0.7,
                timing: "delayed",
                dependencies: ["share_partial_solution"],
            },
        ],
        emergentOutcomes: [
            {
                description: "Solution quality exceeds best individual",
                indicators: ["solution_quality", "collective_accuracy"],
                minimumAgents: 2,
            },
        ],
    }),

    ADAPTIVE_RESILIENCE: createPattern({
        name: "adaptive-resilience",
        description: "System adapts to failures and maintains function",
        requiredCapabilities: ["detect_failure", "compensate"],
        triggerConditions: [
            {
                type: "event",
                condition: { event: "component_failure" },
                description: "System component fails",
            },
        ],
        expectedBehaviors: [
            {
                action: "redistribute_load",
                probability: 0.95,
                timing: "immediate",
            },
            {
                action: "spawn_replacement",
                probability: 0.7,
                timing: "delayed",
            },
            {
                action: "adapt_strategy",
                probability: 0.6,
                timing: "eventual",
            },
        ],
        emergentOutcomes: [
            {
                description: "System maintains performance despite failures",
                indicators: ["uptime", "performance_stability"],
                minimumAgents: 3,
            },
        ],
    }),

    STIGMERGIC_COORDINATION: createPattern({
        name: "stigmergic-coordination",
        description: "Coordination through environmental traces",
        requiredCapabilities: ["leave_trace", "read_trace"],
        triggerConditions: [
            {
                type: "state",
                condition: { shared_environment: true },
                description: "Agents share environment",
            },
        ],
        expectedBehaviors: [
            {
                action: "mark_territory",
                probability: 0.8,
                timing: "immediate",
            },
            {
                action: "follow_trace",
                probability: 0.7,
                timing: "delayed",
            },
            {
                action: "reinforce_path",
                probability: 0.6,
                timing: "eventual",
                dependencies: ["follow_trace"],
            },
        ],
        emergentOutcomes: [
            {
                description: "Optimal paths emerge from collective behavior",
                indicators: ["path_efficiency", "trace_convergence"],
                minimumAgents: 5,
            },
        ],
    }),
};

export class PatternMatcher {
    private observations: ObservedBehavior[] = [];
    private eventPublisher: EventPublisher;

    constructor(eventPublisher: EventPublisher) {
        this.eventPublisher = eventPublisher;
    }

    /**
     * Test if agents exhibit a specific behavior pattern
     */
    async testPattern(
        agents: EmergentAgent[],
        pattern: BehaviorPattern,
        duration = 30000,
    ): Promise<PatternTestResult> {
        // Set up observation
        this.setupObservation(agents);

        // Apply trigger conditions
        await this.applyTriggers(agents, pattern.triggerConditions);

        // Observe for specified duration
        await new Promise(resolve => setTimeout(resolve, duration));

        // Analyze observations
        const result = this.analyzePattern(pattern);

        return result;
    }

    /**
     * Detect any emergent patterns not explicitly defined
     */
    async detectEmergentPatterns(
        agents: EmergentAgent[],
        observationTime: number,
    ): Promise<DetectedPattern[]> {
        this.setupObservation(agents);

        // Observe without intervention
        await new Promise(resolve => setTimeout(resolve, observationTime));

        // Analyze for patterns
        return this.findEmergentPatterns();
    }

    private setupObservation(agents: EmergentAgent[]) {
        agents.forEach(agent => {
            agent.on("action", (action: any) => {
                this.observations.push({
                    agent: agent.id,
                    action: action.type,
                    timestamp: new Date(),
                    metrics: action.metrics || {},
                });
            });
        });
    }

    private async applyTriggers(
        agents: EmergentAgent[],
        triggers: TriggerCondition[],
    ) {
        for (const trigger of triggers) {
            switch (trigger.type) {
                case "event":
                    await this.eventPublisher.publish({
                        type: trigger.condition.event,
                        timestamp: new Date(),
                    });
                    break;

                case "metric":
                    // Simulate metric condition
                    agents.forEach(agent => {
                        agent.updateMetrics(trigger.condition);
                    });
                    break;

                case "state":
                    // Apply state condition
                    agents.forEach(agent => {
                        agent.setState(trigger.condition);
                    });
                    break;

                case "time":
                    // Wait for time condition
                    await new Promise(resolve => 
                        setTimeout(resolve, trigger.condition.delay),
                    );
                    break;
            }
        }
    }

    private analyzePattern(pattern: BehaviorPattern): PatternTestResult {
        const behaviorMatches = this.matchExpectedBehaviors(
            pattern.expectedBehaviors,
        );

        const emergentOutcomes = this.identifyEmergentOutcomes(
            pattern.emergentOutcomes || [],
        );

        const confidence = this.calculateConfidence(
            behaviorMatches,
            pattern.expectedBehaviors,
        );

        return {
            pattern: pattern.name,
            matched: confidence > 0.6,
            behaviors: this.observations,
            emergentOutcomes,
            confidence,
        };
    }

    private matchExpectedBehaviors(
        expected: ExpectedBehavior[],
    ): Map<string, number> {
        const matches = new Map<string, number>();

        for (const behavior of expected) {
            const count = this.observations.filter(
                obs => obs.action === behavior.action,
            ).length;
            
            matches.set(behavior.action, count);
        }

        return matches;
    }

    private identifyEmergentOutcomes(
        possibleOutcomes: EmergentOutcome[],
    ): string[] {
        const identified: string[] = [];

        for (const outcome of possibleOutcomes) {
            if (this.hasEmergentIndicators(outcome)) {
                identified.push(outcome.description);
            }
        }

        return identified;
    }

    private hasEmergentIndicators(outcome: EmergentOutcome): boolean {
        // Check if minimum number of agents involved
        const uniqueAgents = new Set(this.observations.map(o => o.agent));
        if (uniqueAgents.size < outcome.minimumAgents) {
            return false;
        }

        // Check for indicators in observations
        const observationData = JSON.stringify(this.observations);
        return outcome.indicators.some(indicator => 
            observationData.includes(indicator),
        );
    }

    private calculateConfidence(
        matches: Map<string, number>,
        expected: ExpectedBehavior[],
    ): number {
        let totalScore = 0;
        let totalWeight = 0;

        for (const behavior of expected) {
            const observed = matches.get(behavior.action) || 0;
            const weight = behavior.probability;
            
            // Score based on whether behavior occurred
            const score = observed > 0 ? 1 : 0;
            
            totalScore += score * weight;
            totalWeight += weight;
        }

        return totalWeight > 0 ? totalScore / totalWeight : 0;
    }

    private findEmergentPatterns(): DetectedPattern[] {
        const patterns: DetectedPattern[] = [];

        // Group observations by time windows
        const timeWindows = this.groupByTimeWindow(this.observations, 5000);

        // Look for recurring sequences
        const sequences = this.findRecurringSequences(timeWindows);

        // Look for agent clustering
        const clusters = this.findAgentClusters(this.observations);

        // Convert findings to patterns
        sequences.forEach(seq => {
            patterns.push({
                type: "sequence",
                description: `Recurring sequence: ${seq.actions.join(" → ")}`,
                frequency: seq.count,
                confidence: seq.count / timeWindows.length,
            });
        });

        clusters.forEach(cluster => {
            patterns.push({
                type: "clustering",
                description: `Agent cluster: ${cluster.agents.join(", ")}`,
                frequency: cluster.interactions,
                confidence: cluster.cohesion,
            });
        });

        return patterns;
    }

    private groupByTimeWindow(
        observations: ObservedBehavior[],
        windowSize: number,
    ): ObservedBehavior[][] {
        const windows: ObservedBehavior[][] = [];
        let currentWindow: ObservedBehavior[] = [];
        let windowStart = observations[0]?.timestamp.getTime() || 0;

        for (const obs of observations) {
            if (obs.timestamp.getTime() - windowStart > windowSize) {
                windows.push(currentWindow);
                currentWindow = [];
                windowStart = obs.timestamp.getTime();
            }
            currentWindow.push(obs);
        }

        if (currentWindow.length > 0) {
            windows.push(currentWindow);
        }

        return windows;
    }

    private findRecurringSequences(
        timeWindows: ObservedBehavior[][],
    ): RecurringSequence[] {
        const sequenceMap = new Map<string, number>();

        for (const window of timeWindows) {
            const sequence = window.map(o => o.action).join(",");
            sequenceMap.set(sequence, (sequenceMap.get(sequence) || 0) + 1);
        }

        return Array.from(sequenceMap.entries())
            .filter(([_, count]) => count > 1)
            .map(([sequence, count]) => ({
                actions: sequence.split(","),
                count,
            }));
    }

    private findAgentClusters(
        observations: ObservedBehavior[],
    ): AgentCluster[] {
        const interactions = new Map<string, Set<string>>();

        // Build interaction graph
        observations.forEach((obs, i) => {
            const nearbyObs = observations.filter((o, j) => 
                Math.abs(j - i) <= 5 && o.agent !== obs.agent,
            );

            nearbyObs.forEach(nearby => {
                const key = [obs.agent, nearby.agent].sort().join("-");
                if (!interactions.has(key)) {
                    interactions.set(key, new Set());
                }
                interactions.get(key)!.add(obs.action);
            });
        });

        // Find clusters (simplified - just pairs with high interaction)
        const clusters: AgentCluster[] = [];
        interactions.forEach((actions, agentPair) => {
            if (actions.size > 3) {
                const [agent1, agent2] = agentPair.split("-");
                clusters.push({
                    agents: [agent1, agent2],
                    interactions: actions.size,
                    cohesion: actions.size / 10, // Simplified cohesion metric
                });
            }
        });

        return clusters;
    }
}

export interface DetectedPattern {
    type: string;
    description: string;
    frequency: number;
    confidence: number;
}

export interface RecurringSequence {
    actions: string[];
    count: number;
}

export interface AgentCluster {
    agents: string[];
    interactions: number;
    cohesion: number;
}

/**
 * Helper to create a behavior pattern
 */
export function createPattern(config: BehaviorPattern): BehaviorPattern {
    return config;
}

/**
 * Test helpers for common emergent behaviors
 */
export const EmergentBehaviorTests = {
    /**
     * Test if agents self-organize into roles
     */
    async testRoleEmergence(agents: EmergentAgent[]): Promise<boolean> {
        const roles = new Map<string, string>();
        
        // Observe for role differentiation
        await new Promise(resolve => setTimeout(resolve, 10000));
        
        agents.forEach(agent => {
            const primaryAction = agent.getMostFrequentAction();
            roles.set(agent.id, primaryAction);
        });
        
        // Check if roles are differentiated
        const uniqueRoles = new Set(roles.values());
        return uniqueRoles.size > 1 && uniqueRoles.size <= agents.length;
    },

    /**
     * Test if collective decision making emerges
     */
    async testCollectiveDecision(
        agents: EmergentAgent[],
        decision: string,
    ): Promise<boolean> {
        // Present decision to all agents
        await Promise.all(agents.map(agent => 
            agent.considerDecision(decision),
        ));
        
        // Wait for consensus
        await new Promise(resolve => setTimeout(resolve, 5000));
        
        // Check if consensus emerged
        const decisions = agents.map(a => a.getDecision(decision));
        const consensus = decisions.every(d => d === decisions[0]);
        
        return consensus && decisions[0] !== null;
    },

    /**
     * Test if agents develop communication protocols
     */
    async testProtocolEmergence(agents: EmergentAgent[]): Promise<boolean> {
        const initialProtocols = agents.map(a => a.getCommunicationProtocol());
        
        // Allow interaction time
        await new Promise(resolve => setTimeout(resolve, 15000));
        
        const finalProtocols = agents.map(a => a.getCommunicationProtocol());
        
        // Check if protocols converged
        const protocolsChanged = !initialProtocols.every((p, i) => 
            p === finalProtocols[i],
        );
        
        const protocolsAligned = finalProtocols.every(p => 
            p.format === finalProtocols[0].format,
        );
        
        return protocolsChanged && protocolsAligned;
    },
};
