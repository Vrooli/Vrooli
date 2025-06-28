/**
 * Resource Flow Protocol - Emergent Resource Management for Three-Tier Architecture
 * 
 * This module implements the data-driven resource allocation protocol that enables emergent
 * swarm capabilities. Unlike traditional hard-coded resource management, this protocol:
 * 
 * 1. **Data-Driven Configuration**: All resource policies defined in config objects, not code
 * 2. **Emergent Allocation**: Agents can dynamically adjust resource strategies through events
 * 3. **Event-Driven Optimization**: Resource usage patterns trigger optimization events
 * 4. **Self-Improving Flow**: Historical data enables automatic allocation improvements
 * 
 * ## Architecture Integration:
 * 
 * ```
 * SwarmContextManager
 *         ↓ allocates resources through
 * ResourceFlowProtocol
 *         ↓ creates hierarchical allocations for
 * Tier 1 → Tier 2 → Tier 3 Execution Flow
 * ```
 * 
 * ## Emergent Capabilities Enabled:
 * 
 * - **Resource Strategy Agents**: Monitor allocation efficiency and suggest improvements
 * - **Demand Prediction Agents**: Learn usage patterns and pre-allocate resources
 * - **Cost Optimization Agents**: Balance performance vs cost through dynamic strategies
 * - **Failure Recovery Agents**: Detect resource exhaustion and trigger alternative approaches
 * 
 * @see SwarmContextManager - Central state authority that uses this protocol
 * @see /docs/architecture/execution/swarm-state-management-redesign.md - Complete architecture
 */

import { 
    type CoreResourceAllocation, 
    type ExecutionContext,
    type ExecutionResourceUsage,
    type StepExecutionInput,
    type TierExecutionRequest,
    ResourceAggregator,
    generatePK,
} from "@vrooli/shared";
import { type Logger } from "winston";

/**
 * Resource allocation strategy configuration - fully data-driven for emergent capabilities
 * 
 * @deprecated This temporary bridge interface will be replaced by SwarmContext.policy.resource
 * once SwarmContextManager is fully implemented. Do not extend or build dependencies on this.
 */
export interface ResourceAllocationStrategy {
    /** Strategy name for agent-driven optimization */
    name: string;
    
    /** Resource allocation percentages for each tier (data-driven, not hard-coded) */
    tierAllocations: {
        tier1ToTier2: number; // Percentage of swarm resources allocated to routine execution
        tier2ToTier3: number; // Percentage of routine resources allocated to step execution  
    };
    
    /** Dynamic allocation policies that agents can modify through events */
    policies: {
        /** How to handle resource exhaustion */
        exhaustionStrategy: "fail_fast" | "elastic_retry" | "degraded_mode";
        
        /** Whether to allow resource oversubscription for parallel execution */
        allowOversubscription: boolean;
        
        /** Minimum resource buffer to maintain for emergency allocation */
        bufferPercentage: number;
    };
    
    /** Agent-driven optimization hints */
    optimizationHints: {
        /** Priority weightings that optimization agents can adjust */
        priorities: {
            speed: number;
            cost: number; 
            reliability: number;
        };
        
        /** Historical performance data for agent learning */
        performanceHistory?: {
            averageUtilization: number;
            peakUtilization: number;
            commonBottlenecks: string[];
        };
    };
}

/**
 * Resource allocation result with full traceability for emergent learning
 */
export interface ResourceAllocationResult {
    /** The allocated resources */
    allocation: CoreResourceAllocation;
    
    /** Strategy used for this allocation (for agent analysis) */
    strategy: ResourceAllocationStrategy;
    
    /** Hierarchical context for resource flow tracking */
    hierarchy: {
        parentExecutionId?: string;
        parentAllocation?: CoreResourceAllocation;
        allocationPath: string[]; // e.g., ["swarm-123", "routine-456", "step-789"]
    };
    
    /** Metadata for emergent optimization */
    metadata: {
        allocationTimestamp: Date;
        estimatedDuration: number;
        expectedUtilization: number;
        riskFactors: string[];
    };
}

/**
 * Resource Flow Protocol Implementation
 * 
 * Provides data-driven resource allocation that enables emergent swarm capabilities.
 * All resource policies are configurable, allowing agents to optimize allocation
 * strategies based on observed performance patterns.
 */
export class ResourceFlowProtocol {
    private readonly logger: Logger;
    
    constructor(logger: Logger) {
        this.logger = logger;
    }
    
    /**
     * Create a properly formatted Tier 2 → Tier 3 execution request with resource allocation
     * 
     * This method fixes the critical bug in UnifiedRunStateMachine.createTier3ExecutionRequest()
     * by using the correct TierExecutionRequest format expected by TierThreeExecutor.
     * 
     * @param context - Run execution context from Tier 2
     * @param stepInfo - Step information to execute
     * @param parentAllocation - Resource allocation from the routine execution
     * @param strategy - Resource allocation strategy (data-driven for emergent capabilities)
     * @returns Properly formatted TierExecutionRequest for Tier 3
     */
    createTier3ExecutionRequest(
        context: any, // @deprecated Use proper RunExecutionContext type when SwarmContext is ready
        stepInfo: any, // @deprecated Use proper StepInfo type when SwarmContext is ready  
        parentAllocation: CoreResourceAllocation,
        strategy: ResourceAllocationStrategy = this.getDefaultStrategy(),
    ): TierExecutionRequest<StepExecutionInput> {
        // Extract step execution input in the correct format
        const stepInput: StepExecutionInput = {
            stepId: stepInfo.id || stepInfo.stepId || generatePK().toString(),
            stepType: stepInfo.type || stepInfo.stepType || "tool_call",
            toolName: stepInfo.toolName || stepInfo.tool?.name,
            parameters: stepInfo.parameters || stepInfo.inputs || {},
            strategy: stepInfo.strategy || context.defaultStrategy || "conversational",
        };
        
        // Create execution context in the correct format
        const executionContext: ExecutionContext = {
            executionId: generatePK().toString(),
            parentExecutionId: context.executionId,
            swarmId: context.swarmId,
            userId: context.userId || context.parentContext?.executingAgent || "system",
            timestamp: new Date(),
            correlationId: context.correlationId || generatePK().toString(),
            stepId: stepInput.stepId,
            routineId: context.routineId,
            stepType: stepInput.stepType,
            inputs: stepInput.parameters,
            config: {
                strategy: stepInput.strategy,
                toolName: stepInput.toolName,
                timeout: context.timeout || 30000,
                retryCount: context.retryCount || 3,
            },
        };
        
        // Allocate resources for step execution using data-driven strategy
        const stepAllocation = this.allocateStepResources(
            parentAllocation,
            strategy,
            stepInfo,
        );
        
        // Create the properly formatted request
        const request: TierExecutionRequest<StepExecutionInput> = {
            context: executionContext,
            input: stepInput,
            allocation: stepAllocation,
            options: {
                priority: context.priority || "medium",
                timeout: context.timeout || 30000,
                retryCount: context.retryCount || 3,
                emergentCapabilities: true, // Enable agent-driven optimization
            },
        };
        
        this.logger.debug("[ResourceFlowProtocol] Created Tier 3 execution request", {
            stepId: stepInput.stepId,
            stepType: stepInput.stepType,
            toolName: stepInput.toolName,
            allocation: stepAllocation,
            strategy: strategy.name,
        });
        
        return request;
    }
    
    /**
     * Allocate resources for step execution using data-driven strategy
     * 
     * This implements emergent resource allocation by:
     * 1. Using configurable allocation percentages (not hard-coded)
     * 2. Applying agent-driven optimization hints
     * 3. Creating resource allocations that agents can learn from
     * 
     * @param parentAllocation - Available resources from routine execution
     * @param strategy - Data-driven allocation strategy
     * @param stepInfo - Step information for resource estimation
     * @returns Resource allocation for step execution
     */
    allocateStepResources(
        parentAllocation: CoreResourceAllocation,
        strategy: ResourceAllocationStrategy,
        stepInfo: any, // @deprecated Use proper StepInfo type when SwarmContext is ready
    ): CoreResourceAllocation {
        // Calculate allocation percentage based on data-driven strategy
        const allocationRatio = strategy.tierAllocations.tier2ToTier3;
        
        // Apply buffer to prevent resource exhaustion
        const bufferRatio = 1 - (strategy.policies.bufferPercentage / 100);
        const effectiveRatio = allocationRatio * bufferRatio;
        
        // Estimate resource requirements based on step type and tool
        const resourceMultiplier = this.estimateResourceRequirements(stepInfo, strategy);
        
        // Create child allocation with hierarchical tracking
        const allocation: CoreResourceAllocation = {
            maxCredits: this.calculateCreditsAllocation(
                parentAllocation.maxCredits, 
                effectiveRatio * resourceMultiplier,
            ),
            maxDurationMs: Math.floor(
                parentAllocation.maxDurationMs * effectiveRatio * resourceMultiplier,
            ),
            maxMemoryMB: Math.floor(
                parentAllocation.maxMemoryMB * effectiveRatio * resourceMultiplier,
            ),
            maxConcurrentSteps: Math.min(
                parentAllocation.maxConcurrentSteps,
                strategy.policies.allowOversubscription ? 
                    parentAllocation.maxConcurrentSteps * 2 : 
                    parentAllocation.maxConcurrentSteps,
            ),
        };
        
        // Validate allocation hierarchy
        const validation = ResourceAggregator.validateAllocationHierarchy(allocation, parentAllocation);
        if (!validation.isValid) {
            this.logger.warn("[ResourceFlowProtocol] Resource allocation validation failed", {
                violations: validation.violations,
                parentAllocation,
                childAllocation: allocation,
                strategy: strategy.name,
            });
            
            // Fall back to conservative allocation
            return this.createConservativeAllocation(parentAllocation);
        }
        
        this.logger.debug("[ResourceFlowProtocol] Allocated step resources", {
            parentCredits: parentAllocation.maxCredits,
            allocatedCredits: allocation.maxCredits,
            allocationRatio: effectiveRatio,
            resourceMultiplier,
            strategy: strategy.name,
        });
        
        return allocation;
    }
    
    /**
     * Get default resource allocation strategy
     * 
     * This provides a baseline strategy that agents can optimize through events.
     * All values are configurable to enable emergent improvements.
     * 
     * @deprecated This will be replaced by SwarmContext.policy.resource once SwarmContextManager
     * is implemented. This temporary strategy enables the critical bug fix.
     */
    private getDefaultStrategy(): ResourceAllocationStrategy {
        return {
            name: "default_balanced",
            tierAllocations: {
                tier1ToTier2: 0.8, // 80% of swarm resources to routine execution
                tier2ToTier3: 0.6, // 60% of routine resources to step execution
            },
            policies: {
                exhaustionStrategy: "elastic_retry",
                allowOversubscription: false,
                bufferPercentage: 10, // Reserve 10% for emergency allocation
            },
            optimizationHints: {
                priorities: {
                    speed: 0.4,
                    cost: 0.3,
                    reliability: 0.3,
                },
            },
        };
    }
    
    /**
     * Estimate resource requirements based on step type and tool
     * 
     * Returns a multiplier for resource allocation based on:
     * - Step type complexity (tool_call vs data_transform vs decision_point)
     * - Tool resource requirements (LLM model size, API call complexity)
     * - Historical performance data (when available)
     * 
     * This enables agents to learn optimal resource allocation patterns.
     */
    private estimateResourceRequirements(
        stepInfo: any, // @deprecated Use proper StepInfo when available
        strategy: ResourceAllocationStrategy,
    ): number {
        let multiplier = 1.0;
        
        // Base multiplier by step type (configurable for agent optimization)
        const stepTypeMultipliers: Record<string, number> = {
            "tool_call": 1.2,        // Tool calls typically need more resources
            "api_request": 1.1,      // API requests need network buffer
            "data_transform": 0.8,   // Data transforms are usually lightweight
            "decision_point": 0.6,   // Decision points are quick evaluations
            "loop_iteration": 1.0,   // Standard allocation for loops
            "conditional_branch": 0.7, // Branches are typically quick
            "parallel_execution": 1.5, // Parallel execution needs more resources
            "subroutine_call": 1.3,  // Subroutines can be complex
        };
        
        const stepType = stepInfo.type || stepInfo.stepType || "tool_call";
        multiplier *= stepTypeMultipliers[stepType] || 1.0;
        
        // Tool-specific multipliers (enables agent learning about tool costs)
        if (stepInfo.toolName || stepInfo.tool?.name) {
            const toolName = stepInfo.toolName || stepInfo.tool?.name;
            
            // AI model tools typically need more resources
            if (toolName.includes("gpt") || toolName.includes("claude") || toolName.includes("llm")) {
                multiplier *= 1.4;
            }
            
            // Web scraping and API tools need network buffer
            if (toolName.includes("web") || toolName.includes("api") || toolName.includes("fetch")) {
                multiplier *= 1.2;
            }
            
            // File processing tools need memory buffer  
            if (toolName.includes("file") || toolName.includes("image") || toolName.includes("document")) {
                multiplier *= 1.3;
            }
        }
        
        // Apply optimization hints from strategy
        const { speed, cost, reliability } = strategy.optimizationHints.priorities;
        
        // Higher speed priority = more resource allocation
        if (speed > 0.6) {
            multiplier *= 1.2;
        }
        
        // Higher cost priority = less resource allocation
        if (cost > 0.6) {
            multiplier *= 0.8;
        }
        
        // Higher reliability priority = more resource buffer
        if (reliability > 0.6) {
            multiplier *= 1.1;
        }
        
        // Clamp multiplier to reasonable bounds
        return Math.max(0.3, Math.min(3.0, multiplier));
    }
    
    /**
     * Calculate credits allocation with proper BigInt handling
     */
    private calculateCreditsAllocation(maxCredits: string, ratio: number): string {
        if (maxCredits === "unlimited") {
            return "unlimited";
        }
        
        try {
            // Validate inputs
            if (typeof maxCredits !== "string" || maxCredits.trim() === "") {
                throw new Error("Invalid maxCredits: must be a non-empty string");
            }
            if (typeof ratio !== "number" || ratio < 0 || ratio > 1) {
                throw new Error("Invalid ratio: must be a number between 0 and 1");
            }
            
            const credits = BigInt(maxCredits);
            
            // Use BigInt arithmetic to avoid precision loss
            // Multiply by 1000000 for 6 decimal places, then divide back
            const ratioScale = 1000000n;
            const scaledRatio = BigInt(Math.floor(ratio * Number(ratioScale)));
            const allocated = (credits * scaledRatio) / ratioScale;
            
            return allocated.toString();
        } catch (error) {
            this.logger.error("[ResourceFlowProtocol] Failed to calculate credits allocation", {
                maxCredits,
                ratio,
                error: error instanceof Error ? error.message : String(error),
            });
            // More appropriate fallback - throw instead of silent failure
            throw new Error(`Unable to calculate credits allocation: ${error instanceof Error ? error.message : String(error)}`);
        }
    }
    
    /**
     * Create conservative allocation when validation fails
     */
    private createConservativeAllocation(parentAllocation: CoreResourceAllocation): CoreResourceAllocation {
        return {
            maxCredits: this.calculateCreditsAllocation(parentAllocation.maxCredits, 0.2),
            maxDurationMs: Math.floor(parentAllocation.maxDurationMs * 0.2),
            maxMemoryMB: Math.floor(parentAllocation.maxMemoryMB * 0.2),
            maxConcurrentSteps: 1, // Conservative concurrency
        };
    }
}
