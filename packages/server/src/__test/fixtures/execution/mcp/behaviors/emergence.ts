import { type EmergentAgent } from "../../../../services/execution/cross-cutting/agents/emergentAgent.js";
import { type EventPublisher } from "../../../../services/execution/shared/EventPublisher.js";
import { logger } from "../../../../services/logger.js";

export interface EmergenceResult {
    emerged: boolean;
    capabilities: string[];
    metrics: EmergenceMetrics;
    timeline: EmergenceEvent[];
    duration: number;
}

export interface EmergenceMetrics {
    startCapabilities: number;
    endCapabilities: number;
    interactionCount: number;
    coordinationEvents: number;
    learningRate: number;
}

export interface EmergenceEvent {
    timestamp: Date;
    type: "capability_added" | "behavior_changed" | "coordination_started" | "learning_observed";
    description: string;
    agents: string[];
    metrics?: Record<string, any>;
}

export class EmergenceTester {
    private eventPublisher: EventPublisher;
    private startTime: number;
    private events: EmergenceEvent[] = [];
    private interactionCount = 0;

    constructor(eventPublisher: EventPublisher) {
        this.eventPublisher = eventPublisher;
        this.startTime = Date.now();
    }

    /**
     * Test that capabilities actually emerge from agent interactions
     */
    async testEmergence(
        agents: EmergentAgent[],
        expectedCapability: string,
        options: {
            timeout?: number;
            minInteractions?: number;
            requiredMetrics?: Partial<EmergenceMetrics>;
        } = {},
    ): Promise<EmergenceResult> {
        const { timeout = 60000, minInteractions = 10 } = options;
        const startCapabilities = this.captureCapabilities(agents);

        // Set up monitoring
        this.setupEmergenceMonitoring(agents);

        // Run simulation
        const simulation = await this.runSimulation(agents, {
            timeout,
            minInteractions,
            targetCapability: expectedCapability,
        });

        // Capture final state
        const endCapabilities = this.captureCapabilities(agents);
        const newCapabilities = this.findEmergentCapabilities(startCapabilities, endCapabilities);

        // Calculate metrics
        const metrics: EmergenceMetrics = {
            startCapabilities: startCapabilities.size,
            endCapabilities: endCapabilities.size,
            interactionCount: this.interactionCount,
            coordinationEvents: this.events.filter(e => e.type === "coordination_started").length,
            learningRate: this.calculateLearningRate(this.events),
        };

        return {
            emerged: newCapabilities.has(expectedCapability),
            capabilities: Array.from(newCapabilities),
            metrics,
            timeline: this.events,
            duration: Date.now() - this.startTime,
        };
    }

    /**
     * Test that emergence happens without explicit coordination
     */
    async testTrueEmergence(
        agents: EmergentAgent[],
        scenario: EmergenceScenario,
    ): Promise<boolean> {
        // Ensure no hardcoded coordination
        const hasExplicitCoordination = agents.some(agent => 
            this.hasHardcodedCoordination(agent),
        );
        if (hasExplicitCoordination) {
            throw new Error("Agents have explicit coordination - not true emergence");
        }

        // Run scenario
        const result = await scenario.execute(agents, this.eventPublisher);

        // Verify emergence criteria
        return this.verifyEmergenceCriteria(result, scenario.expectedEmergence);
    }

    private captureCapabilities(agents: EmergentAgent[]): Set<string> {
        const capabilities = new Set<string>();
        agents.forEach(agent => {
            const agentCaps = agent.getCapabilities();
            agentCaps.forEach(cap => capabilities.add(cap));
        });
        return capabilities;
    }

    private findEmergentCapabilities(
        start: Set<string>,
        end: Set<string>,
    ): Set<string> {
        const emergent = new Set<string>();
        end.forEach(cap => {
            if (!start.has(cap)) {
                emergent.add(cap);
            }
        });
        return emergent;
    }

    private setupEmergenceMonitoring(agents: EmergentAgent[]) {
        // Monitor capability changes
        agents.forEach(agent => {
            agent.on("capability:added", (capability: string) => {
                this.recordEvent({
                    type: "capability_added",
                    description: `Agent ${agent.id} gained capability: ${capability}`,
                    agents: [agent.id],
                });
            });

            agent.on("behavior:changed", (behavior: any) => {
                this.recordEvent({
                    type: "behavior_changed",
                    description: `Agent ${agent.id} behavior evolved`,
                    agents: [agent.id],
                    metrics: behavior,
                });
            });
        });

        // Monitor coordination events
        this.eventPublisher.on("coordination:started", (event: any) => {
            this.recordEvent({
                type: "coordination_started",
                description: "Agents began coordinating",
                agents: event.agents,
                metrics: event.metrics,
            });
        });

        // Monitor learning
        this.eventPublisher.on("learning:observed", (event: any) => {
            this.recordEvent({
                type: "learning_observed",
                description: "Learning behavior detected",
                agents: event.agents,
                metrics: event.learningMetrics,
            });
        });
    }

    private async runSimulation(
        agents: EmergentAgent[],
        options: {
            timeout: number;
            minInteractions: number;
            targetCapability: string;
        },
    ): Promise<void> {
        const startTime = Date.now();
        const { timeout, minInteractions, targetCapability } = options;

        while (Date.now() - startTime < timeout) {
            // Simulate agent interactions
            await this.simulateInteraction(agents);
            this.interactionCount++;

            // Check if target capability emerged
            const currentCaps = this.captureCapabilities(agents);
            if (currentCaps.has(targetCapability) && this.interactionCount >= minInteractions) {
                logger.info("Target capability emerged", {
                    capability: targetCapability,
                    interactions: this.interactionCount,
                    duration: Date.now() - startTime,
                });
                break;
            }

            // Small delay to prevent tight loop
            await new Promise(resolve => setTimeout(resolve, 100));
        }
    }

    private async simulateInteraction(agents: EmergentAgent[]) {
        // Randomly select two agents to interact
        const agent1 = agents[Math.floor(Math.random() * agents.length)];
        const agent2 = agents[Math.floor(Math.random() * agents.length)];

        if (agent1 !== agent2) {
            // Simulate information exchange
            await this.eventPublisher.publish({
                type: "agent:interaction",
                source: agent1.id,
                target: agent2.id,
                timestamp: new Date(),
            });
        }
    }

    private recordEvent(event: Omit<EmergenceEvent, "timestamp">) {
        this.events.push({
            ...event,
            timestamp: new Date(),
        });
    }

    private calculateLearningRate(events: EmergenceEvent[]): number {
        const learningEvents = events.filter(e => e.type === "learning_observed");
        if (learningEvents.length < 2) return 0;

        // Calculate rate of learning events over time
        const timespan = events[events.length - 1].timestamp.getTime() - events[0].timestamp.getTime();
        return learningEvents.length / (timespan / 1000 / 60); // Learning events per minute
    }

    private hasHardcodedCoordination(agent: EmergentAgent): boolean {
        // Check if agent has explicit coordination logic
        // This is simplified - in reality would inspect agent configuration
        const config = agent.getConfiguration();
        return config.coordination?.explicit === true;
    }

    private verifyEmergenceCriteria(
        result: any,
        criteria: EmergenceCriteria,
    ): boolean {
        // Verify all emergence criteria are met
        if (criteria.requiredCapabilities) {
            const hasAll = criteria.requiredCapabilities.every(cap => 
                result.capabilities.includes(cap),
            );
            if (!hasAll) return false;
        }

        if (criteria.minInteractions && result.interactionCount < criteria.minInteractions) {
            return false;
        }

        if (criteria.maxDuration && result.duration > criteria.maxDuration) {
            return false;
        }

        if (criteria.behaviorPattern) {
            return this.matchesBehaviorPattern(result.timeline, criteria.behaviorPattern);
        }

        return true;
    }

    private matchesBehaviorPattern(
        timeline: EmergenceEvent[],
        pattern: string,
    ): boolean {
        // Simple pattern matching - could be more sophisticated
        const eventSequence = timeline.map(e => e.type).join(" → ");
        return eventSequence.includes(pattern);
    }
}

export interface EmergenceScenario {
    name: string;
    description: string;
    execute: (agents: EmergentAgent[], publisher: EventPublisher) => Promise<any>;
    expectedEmergence: EmergenceCriteria;
}

export interface EmergenceCriteria {
    requiredCapabilities?: string[];
    minInteractions?: number;
    maxDuration?: number;
    behaviorPattern?: string;
}

/**
 * Create a test scenario for emergent behavior
 */
export function createEmergenceScenario(
    name: string,
    setup: () => EmergentAgent[],
    trigger: (agents: EmergentAgent[]) => Promise<void>,
    verify: (result: EmergenceResult) => boolean,
): EmergenceScenario {
    return {
        name,
        description: `Test emergent behavior: ${name}`,
        execute: async (agents, publisher) => {
            // Apply trigger condition
            await trigger(agents);

            // Test emergence
            const tester = new EmergenceTester(publisher);
            const result = await tester.testEmergence(agents, name);

            // Verify specific conditions
            if (!verify(result)) {
                throw new Error(`Emergence verification failed for: ${name}`);
            }

            return result;
        },
        expectedEmergence: {
            requiredCapabilities: [name],
            minInteractions: 5,
        },
    };
}
