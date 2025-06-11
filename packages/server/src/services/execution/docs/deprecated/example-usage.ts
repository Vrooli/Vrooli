/**
 * Example usage of the Resource Monitoring System
 * 
 * DEPRECATED: Resource allocation is now handled by individual tier managers.
 * This file demonstrates resource monitoring capabilities only.
 * 
 * For resource allocation:
 * - Tier 1: Use tier1.organization.ResourceManager for swarm resources
 * - Tier 2: Resources managed through RunStateMachine
 * - Tier 3: Use tier3.engine.ResourceManager for step execution
 */

import { logger } from "../../../../events/logger.js";
import { getExecutionArchitecture } from "../executionArchitecture.js";
import {
    AllocationPriority,
    LimitScope,
    ResourceType,
    generatePk,
} from "@vrooli/shared";

/**
 * Example: Setting up and using resource management
 */
async function demonstrateResourceManagement() {
    // 1. Initialize the execution architecture
    const architecture = await getExecutionArchitecture({
        useRedis: false, // Use in-memory for demo
        telemetryEnabled: true,
    });
    
    // 2. Get the unified resource manager
    const resourceManager = architecture.getResourceManager();
    
    // 3. Set resource limits for a user
    await resourceManager.setResourceLimits({
        id: generatePk(),
        scope: LimitScope.USER,
        scopeId: "user123",
        limits: [
            {
                resourceType: ResourceType.CREDITS,
                limit: 10000,
                rollover: false,
            },
            {
                resourceType: ResourceType.TOKENS,
                limit: 50000,
                period: {
                    unit: "hour",
                    value: 1,
                },
                rollover: true,
            },
            {
                resourceType: ResourceType.API_CALLS,
                limit: 1000,
                period: {
                    unit: "minute",
                    value: 1,
                },
                rollover: false,
            },
        ],
        enforcement: {
            mode: "hard",
            overageAllowed: false,
        },
        notifications: [
            {
                threshold: 80, // Alert at 80% usage
                channel: "event",
                recipients: ["system"],
            },
        ],
    });
    
    // 4. Allocate resources for a swarm execution
    const swarmAllocation = await resourceManager.allocateResources(
        {
            id: generatePk(),
            requesterId: "swarm123",
            requesterType: "swarm",
            resources: [
                {
                    resourceId: ResourceType.CREDITS,
                    amount: 1000,
                    required: true,
                },
                {
                    resourceId: ResourceType.TOKENS,
                    amount: 10000,
                    required: true,
                },
            ],
            priority: AllocationPriority.HIGH,
            duration: 3600, // 1 hour
            metadata: {
                purpose: "Document analysis swarm",
            },
        },
        {
            userId: "user123",
            swarmId: "swarm123",
            priority: AllocationPriority.HIGH,
            purpose: "Analyze technical documentation",
        },
    );
    
    logger.info("Swarm resource allocation", {
        allocationId: swarmAllocation.allocationId,
        status: swarmAllocation.status,
        allocated: swarmAllocation.allocatedResources,
    });
    
    // 5. Track usage during execution
    await resourceManager.trackUsage(swarmAllocation.allocationId, {
        id: generatePk(),
        allocationId: swarmAllocation.allocationId,
        resourceId: ResourceType.TOKENS,
        startTime: new Date(),
        consumed: 5000,
        remaining: 5000,
        cost: 0.1,
        metadata: {
            model: "gpt-4",
            steps: 3,
        },
    });
    
    // 6. Get usage report
    const usageReport = await resourceManager.getUsageReport(
        LimitScope.USER,
        "user123",
    );
    
    logger.info("User resource usage report", {
        period: usageReport.period,
        usage: usageReport.usage,
        costs: usageReport.costs,
        efficiency: usageReport.efficiency,
    });
    
    // 7. Get optimization suggestions
    const optimizations = await resourceManager.getOptimizationSuggestions(
        LimitScope.SWARM,
        "swarm123",
    );
    
    logger.info("Resource optimization suggestions", {
        count: optimizations.length,
        suggestions: optimizations.map(s => ({
            type: s.type,
            resource: s.targetResource,
            savings: s.projectedSavings,
            risk: s.risk,
        })),
    });
    
    // 8. Handle resource conflict (simulated)
    const conflict = {
        id: generatePk(),
        timestamp: new Date(),
        type: "contention" as const,
        resources: [ResourceType.TOKENS],
        requesters: [swarmAllocation.allocationId],
        resolved: false,
    };
    
    const resolution = await resourceManager.resolveConflict(conflict);
    logger.info("Resource conflict resolution", {
        conflictId: conflict.id,
        resolution: resolution,
    });
    
    // 9. Release resources when done
    await resourceManager.releaseResources(
        swarmAllocation.allocationId,
        swarmAllocation.token,
    );
    
    logger.info("Resources released", {
        allocationId: swarmAllocation.allocationId,
    });
    
    // 10. Stop the architecture
    await architecture.stop();
}

/**
 * Example: Hierarchical resource allocation
 */
async function demonstrateHierarchicalAllocation() {
    const architecture = await getExecutionArchitecture();
    const resourceManager = architecture.getResourceManager();
    
    // Set limits at different scopes
    const scopes = [
        { scope: LimitScope.USER, id: "user456", credits: 100000 },
        { scope: LimitScope.SWARM, id: "swarm456", credits: 50000 },
        { scope: LimitScope.RUN, id: "run456", credits: 10000 },
        { scope: LimitScope.STEP, id: "step456", credits: 1000 },
    ];
    
    for (const { scope, id, credits } of scopes) {
        await resourceManager.setResourceLimits({
            id: generatePk(),
            scope,
            scopeId: id,
            limits: [{
                resourceType: ResourceType.CREDITS,
                limit: credits,
                rollover: false,
            }],
            enforcement: {
                mode: "hard",
                overageAllowed: false,
            },
            notifications: [],
        });
    }
    
    // Allocate at step level - checks all parent scopes
    const stepAllocation = await resourceManager.allocateResources(
        {
            id: generatePk(),
            requesterId: "step456",
            requesterType: "step",
            resources: [{
                resourceId: ResourceType.CREDITS,
                amount: 500,
                required: true,
            }],
            priority: AllocationPriority.NORMAL,
            metadata: {},
        },
        {
            userId: "user456",
            swarmId: "swarm456",
            runId: "run456",
            stepId: "step456",
            priority: AllocationPriority.NORMAL,
            purpose: "Execute API call",
        },
    );
    
    logger.info("Hierarchical allocation result", {
        status: stepAllocation.status,
        allocated: stepAllocation.allocatedResources,
    });
    
    await architecture.stop();
}

/**
 * Example: Resource monitoring and alerts
 */
async function demonstrateResourceMonitoring() {
    const architecture = await getExecutionArchitecture();
    const resourceManager = architecture.getResourceManager();
    const eventBus = architecture.getEventBus();
    
    // Subscribe to resource alerts
    await eventBus.subscribe({
        id: generatePk(),
        eventType: "RESOURCE_ALERT",
        handler: "handleResourceAlert",
        filters: [],
        config: {
            maxRetries: 3,
            retryDelay: 1000,
        },
    });
    
    // Set limits with notification thresholds
    await resourceManager.setResourceLimits({
        id: generatePk(),
        scope: LimitScope.USER,
        scopeId: "user789",
        limits: [{
            resourceType: ResourceType.CREDITS,
            limit: 1000,
            rollover: false,
        }],
        enforcement: {
            mode: "soft",
            gracePeriod: 300, // 5 minutes
            overageAllowed: true,
            overageMultiplier: 2, // 2x cost for overage
        },
        notifications: [
            {
                threshold: 50,
                channel: "event",
                recipients: ["system"],
                message: "50% of credits consumed",
            },
            {
                threshold: 80,
                channel: "event",
                recipients: ["system"],
                message: "80% of credits consumed - consider optimization",
            },
            {
                threshold: 100,
                channel: "event",
                recipients: ["system"],
                message: "Credit limit reached - overage charges apply",
            },
        ],
    });
    
    // Simulate gradual resource consumption
    const allocationResult = await resourceManager.allocateResources(
        {
            id: generatePk(),
            requesterId: "run789",
            requesterType: "run",
            resources: [{
                resourceId: ResourceType.CREDITS,
                amount: 1200, // Exceeds limit
                required: true,
            }],
            priority: AllocationPriority.HIGH,
            metadata: {},
        },
        {
            userId: "user789",
            runId: "run789",
            priority: AllocationPriority.HIGH,
            purpose: "Large document processing",
        },
    );
    
    // Track usage that exceeds allocation
    if (allocationResult.status === "allocated") {
        await resourceManager.trackUsage(allocationResult.allocationId, {
            id: generatePk(),
            allocationId: allocationResult.allocationId,
            resourceId: ResourceType.CREDITS,
            startTime: new Date(),
            consumed: 1100, // Overage
            remaining: -100,
            cost: 1.2, // Including overage cost
            metadata: {},
        });
    }
    
    await architecture.stop();
}

// Run examples
if (require.main === module) {
    (async () => {
        try {
            logger.info("=== Basic Resource Management Demo ===");
            await demonstrateResourceManagement();
            
            logger.info("\n=== Hierarchical Allocation Demo ===");
            await demonstrateHierarchicalAllocation();
            
            logger.info("\n=== Resource Monitoring Demo ===");
            await demonstrateResourceMonitoring();
            
        } catch (error) {
            logger.error("Demo failed", error);
        }
    })();
}