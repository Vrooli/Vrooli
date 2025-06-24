import { type Logger } from "winston";
import { type IntelligentEvent, type AgentResponse } from "../events/eventBus.js";

/**
 * Barrier Synchronizer - Safety-critical event coordination
 * 
 * Implements barrier synchronization for safety-critical operations that require
 * approval from specialized safety agents before execution can proceed.
 */
export class BarrierSynchronizer {
    private readonly logger: Logger;
    private readonly eventBus: any; // Will be properly typed
    private readonly pendingBarriers = new Map<string, BarrierState>();
    private readonly safetyAgents = new Map<string, SafetyAgent>();
    
    // Barrier configuration
    private readonly DEFAULT_TIMEOUT_MS = 5000; // 5 seconds
    private readonly DEFAULT_QUORUM = 1; // At least 1 OK response required
    private readonly MAX_BARRIER_WAIT_TIME = 30000; // 30 seconds max

    constructor(eventBus: any, logger?: Logger) {
        this.eventBus = eventBus;
        this.logger = logger || console as any;
        
        // Register default safety agents
        this.registerDefaultSafetyAgents();
    }

    /**
     * Synchronize on a barrier event - wait for safety agent approval
     */
    async synchronize(event: IntelligentEvent): Promise<BarrierResult> {
        const barrierId = `barrier_${event.id}_${Date.now()}`;
        const timeout = event.barrierTimeout || this.DEFAULT_TIMEOUT_MS;
        const quorum = event.barrierQuorum || this.DEFAULT_QUORUM;

        this.logger.info("[BarrierSynchronizer] Starting barrier synchronization", {
            barrierId,
            eventId: event.id,
            eventType: event.type,
            timeout,
            quorum,
        });

        try {
            // Create barrier state
            const barrierState: BarrierState = {
                id: barrierId,
                eventId: event.id,
                event,
                status: "WAITING",
                createdAt: new Date(),
                timeout,
                quorum,
                responses: new Map(),
                resolvedAt: null,
            };

            this.pendingBarriers.set(barrierId, barrierState);

            // Find relevant safety agents
            const relevantAgents = this.findRelevantSafetyAgents(event);
            
            if (relevantAgents.length === 0) {
                this.logger.warn("[BarrierSynchronizer] No safety agents found for event", {
                    barrierId,
                    eventType: event.type,
                });
                
                // Default to ALLOW if no agents are available
                return this.resolveBarrier(barrierId, "OK", "No safety agents available - default allow");
            }

            // Request approval from safety agents
            const approvalPromises = relevantAgents.map(agent => 
                this.requestAgentApproval(barrierId, event, agent),
            );

            // Wait for responses with timeout
            const result = await Promise.race([
                this.waitForQuorum(barrierId, quorum),
                this.waitForTimeout(barrierId, timeout),
            ]);

            return result;

        } catch (error) {
            this.logger.error("[BarrierSynchronizer] Barrier synchronization failed", {
                barrierId,
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                status: "ALARM",
                reason: `Barrier synchronization error: ${error instanceof Error ? error.message : String(error)}`,
                agentResponses: [],
                resolvedAt: new Date(),
            };
        } finally {
            // Clean up barrier state
            this.pendingBarriers.delete(barrierId);
        }
    }

    /**
     * Register a safety agent
     */
    registerSafetyAgent(agent: SafetyAgent): void {
        this.safetyAgents.set(agent.id, agent);
        
        this.logger.info("[BarrierSynchronizer] Registered safety agent", {
            agentId: agent.id,
            capabilities: agent.capabilities,
            domains: agent.domains,
        });
    }

    /**
     * Process agent response to barrier
     */
    async processAgentResponse(
        barrierId: string,
        agentId: string,
        response: AgentResponse,
    ): Promise<void> {
        const barrier = this.pendingBarriers.get(barrierId);
        if (!barrier) {
            this.logger.warn("[BarrierSynchronizer] Response for unknown barrier", {
                barrierId,
                agentId,
            });
            return;
        }

        if (barrier.status !== "WAITING") {
            this.logger.debug("[BarrierSynchronizer] Response for resolved barrier", {
                barrierId,
                agentId,
                barrierStatus: barrier.status,
            });
            return;
        }

        // Store agent response
        barrier.responses.set(agentId, {
            agentId,
            response,
            timestamp: new Date(),
        });

        this.logger.debug("[BarrierSynchronizer] Received agent response", {
            barrierId,
            agentId,
            status: response.status,
            confidence: response.confidence,
        });

        // Check if we should resolve the barrier
        await this.checkBarrierResolution(barrierId);
    }

    /**
     * Get barrier status
     */
    getBarrierStatus(barrierId: string): BarrierState | null {
        return this.pendingBarriers.get(barrierId) || null;
    }

    /**
     * Get all pending barriers
     */
    getPendingBarriers(): BarrierState[] {
        return Array.from(this.pendingBarriers.values());
    }

    /**
     * Private helper methods
     */
    private findRelevantSafetyAgents(event: IntelligentEvent): SafetyAgent[] {
        const relevantAgents: SafetyAgent[] = [];

        for (const agent of this.safetyAgents.values()) {
            if (this.agentCanHandleEvent(agent, event)) {
                relevantAgents.push(agent);
            }
        }

        // Sort by priority (higher priority first)
        return relevantAgents.sort((a, b) => b.priority - a.priority);
    }

    private agentCanHandleEvent(agent: SafetyAgent, event: IntelligentEvent): boolean {
        // Check if agent handles this event type
        const handlesType = agent.eventPatterns.some(pattern => {
            if (pattern.includes("*")) {
                const regex = new RegExp(pattern.replace(/\\*/g, ".*"));
                return regex.test(event.type);
            }
            return event.type === pattern;
        });

        if (!handlesType) return false;

        // Check domain expertise
        if (agent.domains.length > 0) {
            const eventDomain = this.extractEventDomain(event);
            if (!agent.domains.includes(eventDomain)) return false;
        }

        // Check security level
        if (event.securityLevel && agent.minSecurityLevel) {
            const securityLevels = ["public", "internal", "confidential", "secret"];
            const eventLevel = securityLevels.indexOf(event.securityLevel);
            const agentLevel = securityLevels.indexOf(agent.minSecurityLevel);
            if (eventLevel < agentLevel) return false;
        }

        return true;
    }

    private extractEventDomain(event: IntelligentEvent): string {
        // Extract domain from event type or data
        if (event.type.includes("financial")) return "financial";
        if (event.type.includes("medical") || event.type.includes("health")) return "medical";
        if (event.type.includes("security") || event.type.includes("safety")) return "security";
        if (event.type.includes("compliance")) return "compliance";
        
        return "general";
    }

    private async requestAgentApproval(
        barrierId: string,
        event: IntelligentEvent,
        agent: SafetyAgent,
    ): Promise<void> {
        try {
            this.logger.debug("[BarrierSynchronizer] Requesting approval from agent", {
                barrierId,
                agentId: agent.id,
                eventType: event.type,
            });

            // Create approval request context
            const approvalContext = {
                barrierId,
                event,
                requiredCapabilities: agent.capabilities,
                timeoutAt: new Date(Date.now() + event.barrierTimeout!),
            };

            // Execute agent's approval handler
            const response = await agent.approvalHandler(approvalContext);

            // Process the response
            await this.processAgentResponse(barrierId, agent.id, response);

        } catch (error) {
            this.logger.error("[BarrierSynchronizer] Agent approval request failed", {
                barrierId,
                agentId: agent.id,
                error: error instanceof Error ? error.message : String(error),
            });

            // Treat agent failure as ALARM
            await this.processAgentResponse(barrierId, agent.id, {
                status: "ALARM",
                confidence: 0,
                reasoning: `Agent approval failed: ${error instanceof Error ? error.message : String(error)}`,
            });
        }
    }

    private async waitForQuorum(barrierId: string, quorum: number): Promise<BarrierResult> {
        return new Promise((resolve) => {
            const checkQuorum = () => {
                const barrier = this.pendingBarriers.get(barrierId);
                if (!barrier || barrier.status !== "WAITING") {
                    return;
                }

                const responses = Array.from(barrier.responses.values());
                const okResponses = responses.filter(r => r.response.status === "OK");
                const alarmResponses = responses.filter(r => r.response.status === "ALARM");

                // If we have enough OK responses, approve
                if (okResponses.length >= quorum) {
                    resolve(this.resolveBarrier(barrierId, "OK", "Quorum reached with OK responses"));
                    return;
                }

                // If any agent raises an alarm, deny immediately
                if (alarmResponses.length > 0) {
                    const alarmReasons = alarmResponses
                        .map(r => r.response.reasoning)
                        .filter(Boolean)
                        .join("; ");
                    
                    resolve(this.resolveBarrier(
                        barrierId, 
                        "ALARM", 
                        `Safety alarm raised: ${alarmReasons}`,
                    ));
                    return;
                }

                // Continue waiting
                setTimeout(checkQuorum, 100); // Check every 100ms
            };

            checkQuorum();
        });
    }

    private async waitForTimeout(barrierId: string, timeout: number): Promise<BarrierResult> {
        return new Promise((resolve) => {
            setTimeout(() => {
                const barrier = this.pendingBarriers.get(barrierId);
                if (barrier && barrier.status === "WAITING") {
                    resolve(this.resolveBarrier(
                        barrierId, 
                        "ALARM", 
                        `Barrier timeout after ${timeout}ms - safety approval required`,
                    ));
                }
            }, timeout);
        });
    }

    private async checkBarrierResolution(barrierId: string): Promise<void> {
        const barrier = this.pendingBarriers.get(barrierId);
        if (!barrier || barrier.status !== "WAITING") {
            return;
        }

        const responses = Array.from(barrier.responses.values());
        const okResponses = responses.filter(r => r.response.status === "OK");
        const alarmResponses = responses.filter(r => r.response.status === "ALARM");

        // Immediate alarm if any agent raises alarm
        if (alarmResponses.length > 0) {
            const alarmReasons = alarmResponses
                .map(r => r.response.reasoning)
                .filter(Boolean)
                .join("; ");
            
            this.resolveBarrier(barrierId, "ALARM", `Safety alarm: ${alarmReasons}`);
            return;
        }

        // Approve if quorum reached
        if (okResponses.length >= barrier.quorum) {
            this.resolveBarrier(barrierId, "OK", "Safety approval granted");
            return;
        }
    }

    private resolveBarrier(
        barrierId: string,
        status: "OK" | "ALARM",
        reason: string,
    ): BarrierResult {
        const barrier = this.pendingBarriers.get(barrierId);
        if (barrier) {
            barrier.status = status;
            barrier.resolvedAt = new Date();
        }

        const result: BarrierResult = {
            status,
            reason,
            agentResponses: barrier ? Array.from(barrier.responses.values()) : [],
            resolvedAt: new Date(),
        };

        this.logger.info("[BarrierSynchronizer] Barrier resolved", {
            barrierId,
            status,
            reason,
            responseCount: result.agentResponses.length,
        });

        return result;
    }

    private registerDefaultSafetyAgents(): void {
        // Financial Security Agent
        this.registerSafetyAgent({
            id: "financial_safety_agent",
            name: "Financial Safety Agent",
            capabilities: ["financial_analysis", "fraud_detection", "compliance_checking"],
            domains: ["financial"],
            eventPatterns: ["tool/called/financial/*", "transaction/*", "safety/pre_action"],
            priority: 10,
            minSecurityLevel: "internal",
            approvalHandler: async (context) => {
                // Basic financial safety checks
                const event = context.event;
                
                if (event.type.includes("financial") && event.data) {
                    const amount = this.extractAmount(event.data);
                    
                    // Flag large transactions
                    if (amount && amount > 100000) {
                        return {
                            status: "ALARM",
                            confidence: 0.9,
                            reasoning: `Large financial transaction detected: $${amount}`,
                            suggestedActions: ["manual_review", "compliance_check"],
                        };
                    }
                }
                
                return {
                    status: "OK",
                    confidence: 0.8,
                    reasoning: "Financial safety check passed",
                };
            },
        });

        // General Security Agent
        this.registerSafetyAgent({
            id: "general_security_agent",
            name: "General Security Agent",
            capabilities: ["security_analysis", "threat_detection"],
            domains: ["security", "general"],
            eventPatterns: ["safety/pre_action", "security/*", "threat/*"],
            priority: 5,
            minSecurityLevel: "public",
            approvalHandler: async (context) => {
                const event = context.event;
                
                // Check for high-risk security levels
                if (event.securityLevel === "secret" || event.securityLevel === "confidential") {
                    return {
                        status: "ALARM",
                        confidence: 0.95,
                        reasoning: "High security level requires manual approval",
                        suggestedActions: ["manual_security_review"],
                    };
                }
                
                return {
                    status: "OK",
                    confidence: 0.7,
                    reasoning: "General security check passed",
                };
            },
        });
    }

    private extractAmount(data: any): number | null {
        if (typeof data === "object" && data !== null) {
            // Look for common amount fields
            const amountFields = ["amount", "value", "cost", "price", "total"];
            
            for (const field of amountFields) {
                if (field in data) {
                    const value = data[field];
                    if (typeof value === "number") return value;
                    if (typeof value === "string") {
                        const parsed = parseFloat(value.replace(/[^0-9.-]/g, ""));
                        if (!isNaN(parsed)) return parsed;
                    }
                }
            }
        }
        
        return null;
    }
}

/**
 * Supporting interfaces
 */
interface BarrierState {
    id: string;
    eventId: string;
    event: IntelligentEvent;
    status: "WAITING" | "OK" | "ALARM";
    createdAt: Date;
    resolvedAt: Date | null;
    timeout: number;
    quorum: number;
    responses: Map<string, AgentBarrierResponse>;
}

interface AgentBarrierResponse {
    agentId: string;
    response: AgentResponse;
    timestamp: Date;
}

export interface BarrierResult {
    status: "OK" | "ALARM";
    reason: string;
    agentResponses: AgentBarrierResponse[];
    resolvedAt: Date;
}

export interface SafetyAgent {
    id: string;
    name: string;
    capabilities: string[];
    domains: string[];
    eventPatterns: string[];
    priority: number;
    minSecurityLevel?: "public" | "internal" | "confidential" | "secret";
    approvalHandler: (context: ApprovalContext) => Promise<AgentResponse>;
}

interface ApprovalContext {
    barrierId: string;
    event: IntelligentEvent;
    requiredCapabilities: string[];
    timeoutAt: Date;
}
