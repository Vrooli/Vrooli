import {
    generatePK,
    type ExecutionStrategy,
    type TierCapabilities,
    type TierCommunicationInterface,
    type UnifiedExecutorConfig,
} from "@vrooli/shared";
import { type Logger } from "winston";
import { logger } from "../../../events/logger.js";
import { CacheService } from "../../../redisConn.js";
import { CachedConversationStateStore, PrismaChatStore } from "../../conversation/chatStore.js";
import { getUnifiedEventSystem } from "../../events/initialization/eventSystemService.js";
import { type IEventBus } from "../../events/types.js";
import { ContextSubscriptionManager } from "../shared/ContextSubscriptionManager.js";
import { SwarmContextManager, type ISwarmContextManager } from "../shared/SwarmContextManager.js";
import { SwarmCoordinator } from "../tier1/coordination/swarmCoordinator.js";
import { createConversationBridge } from "../tier1/intelligence/conversationBridge.js";
import { ResourceManager as Tier1ResourceManager } from "../tier1/organization/resourceManager.js";
import { InMemoryRunStateStore } from "../tier2/state/inMemoryRunStateStore.js";
import { RedisRunStateStore, type IRunStateStore } from "../tier2/state/runStateStore.js";
import { TierTwoOrchestrator } from "../tier2/tierTwoOrchestrator.js";
import { ResourceManager as Tier3ResourceManager } from "../tier3/engine/resourceManager.js";
import { UnifiedExecutor } from "../tier3/engine/unifiedExecutor.js";
import { IntegratedToolRegistry } from "./mcp/toolRegistry.js";

/**
 * Strategy factory interface for creating execution strategies
 */
interface StrategyFactory {
    createStrategy: (type: string) => ExecutionStrategy;
}

/**
 * Architecture configuration interface
 */
interface ExecutionArchitectureConfig {
    tier1?: Record<string, unknown>;
    tier2?: Record<string, unknown>;
    tier3?: Partial<UnifiedExecutorConfig>;
}

/**
 * Configuration options for ExecutionArchitecture
 */
export interface ExecutionArchitectureOptions {
    /** Use Redis for state storage (production) or in-memory (development) */
    useRedis?: boolean;
    // telemetryEnabled removed - monitoring now emergent
    // Rolling history options removed - monitoring now emergent
    /** Custom logger instance */
    logger?: Logger;
    /** Custom configuration overrides */
    config?: Partial<ExecutionArchitectureConfig>;
}

/**
 * ExecutionArchitecture Factory
 * 
 * This is the central orchestration point for Vrooli's three-tier execution system.
 * It properly wires together all tiers with dependency injection, creating a unified
 * execution architecture that external systems can use.
 * 
 * The architecture consists of:
 * - Tier 1: Coordination Intelligence (SwarmCoordinator with SwarmContextManager integration)
 * - Tier 2: Process Intelligence (Routine execution)
 * - Tier 3: Execution Intelligence (Step execution)
 * 
 * ## State Management Migration
 * 
 * This factory now supports both legacy and modern state management:
 * - **Modern (Recommended)**: SwarmContextManager + ContextSubscriptionManager for live updates
 * - **Legacy**: Traditional state stores (deprecated - used only for backward compatibility)
 * 
 * Each tier communicates through the standardized TierCommunicationInterface,
 * enabling clean delegation and separation of concerns.
 */
export class ExecutionArchitecture {
    private tier1: TierCommunicationInterface | null = null;
    private tier2: TierCommunicationInterface | null = null;
    private tier3: TierCommunicationInterface | null = null;

    private eventBus: IEventBus | null = null;
    private runStateStore: IRunStateStore | null = null;
    private toolRegistry: IntegratedToolRegistry | null = null;
    private conversationStore: CachedConversationStateStore | null = null;

    // Modern state management components
    private swarmContextManager: ISwarmContextManager | null = null;
    private contextSubscriptionManager: ContextSubscriptionManager | null = null;

    // Resource management
    private tier1ResourceManager: Tier1ResourceManager | null = null;
    private tier3ResourceManager: Tier3ResourceManager | null = null;
    // resourceMonitor removed - monitoring now handled by emergent agents

    private readonly logger: Logger;
    private readonly options: ExecutionArchitectureOptions;
    private initialized = false;

    constructor(options: ExecutionArchitectureOptions = {}) {
        this.options = {
            useRedis: process.env.NODE_ENV === "production",
            ...options,
        };

        this.logger = options.logger || logger;
    }

    /**
     * Creates and initializes the complete execution architecture
     * This is the main factory method that wires everything together
     */
    static async create(options: ExecutionArchitectureOptions = {}): Promise<ExecutionArchitecture> {
        const architecture = new ExecutionArchitecture(options);
        await architecture.initialize();
        return architecture;
    }

    /**
     * Initializes all components with proper dependency injection
     */
    private async initialize(): Promise<void> {
        if (this.initialized) {
            throw new Error("ExecutionArchitecture already initialized");
        }

        this.logger.info("[ExecutionArchitecture] Initializing execution architecture", {
            useRedis: this.options.useRedis,
        });

        try {
            // 1. Initialize shared services
            await this.initializeSharedServices();

            // 2. Initialize Tier 3 (no dependencies on other tiers)
            await this.initializeTier3();

            // 3. Initialize Tier 2 (depends on Tier 3)
            await this.initializeTier2();

            // 4. Initialize Tier 1 (depends on Tier 2)
            await this.initializeTier1();

            // 5. Wire up event listeners between tiers
            await this.wireEventListeners();

            this.initialized = true;

            this.logger.info("[ExecutionArchitecture] Initialization complete", {
                tier1Ready: !!this.tier1,
                tier2Ready: !!this.tier2,
                tier3Ready: !!this.tier3,
                modernStateManagement: !!this.swarmContextManager,
            });

        } catch (error) {
            this.logger.error("[ExecutionArchitecture] Initialization failed", {
                error: error instanceof Error ? error.message : String(error),
            });

            // Clean up any partially initialized components
            await this.cleanup();

            throw error;
        }
    }

    /**
     * Initialize shared services (EventBus, State stores, Tool Registry)
     */
    private async initializeSharedServices(): Promise<void> {
        this.logger.debug("[ExecutionArchitecture] Initializing shared services");

        // Initialize EventBus
        this.eventBus = getUnifiedEventSystem();
        if (this.eventBus) {
            await this.eventBus.start();
        } else {
            this.logger.warn("[ExecutionArchitecture] Unified event system not available");
        }

        // Initialize modern state management components
        const redis = await CacheService.get().raw();
        this.contextSubscriptionManager = new ContextSubscriptionManager(redis, this.logger);
        await this.contextSubscriptionManager.initialize();

        this.swarmContextManager = new SwarmContextManager(redis, this.logger);
        await this.swarmContextManager.initialize();

        // Initialize run state store (still using existing implementation during transition)
        if (this.options.useRedis) {
            this.runStateStore = new RedisRunStateStore();
        } else {
            this.runStateStore = new InMemoryRunStateStore();
        }

        // Initialize conversation store with Prisma-backed implementation
        const chatStore = new PrismaChatStore();
        this.conversationStore = new CachedConversationStateStore(chatStore);

        // Initialize integrated tool registry
        this.toolRegistry = IntegratedToolRegistry.getInstance(this.logger, this.conversationStore);

        // Rolling history removed - monitoring now handled by emergent agents

        // Initialize monitoring tools with system user
        if (this.toolRegistry && this.eventBus) {
            const systemUser = {
                id: "system",
                languages: ["en"],
            };

            this.toolRegistry.initializeMonitoringTools(
                systemUser,
                this.eventBus,
            );
        }

        // Initialize resource managers
        this.tier1ResourceManager = new Tier1ResourceManager(this.logger, this.eventBus!);
        this.tier3ResourceManager = new Tier3ResourceManager(this.logger, this.eventBus!);

        this.logger.debug("[ExecutionArchitecture] Shared services initialized");
    }

    /**
     * Initialize Tier 3 - Execution Intelligence
     */
    private async initializeTier3(): Promise<void> {
        this.logger.debug("[ExecutionArchitecture] Initializing Tier 3");

        // Configure UnifiedExecutor
        const executorConfig: UnifiedExecutorConfig = {
            strategyFactory: {
                defaultStrategy: "CONVERSATIONAL" as any,
                fallbackChain: ["REASONING", "DETERMINISTIC"] as any[],
                adaptationEnabled: true,
                learningRate: 0.1,
            },
            resourceLimits: {
                tokens: 100000,
                apiCalls: 1000,
                computeTime: 3600000,
                memory: 1024,
                cost: 100,
            },
            sandboxEnabled: false,
            telemetryEnabled: true,
            ...this.options.config?.tier3,
        };

        // Create Tier 3 with integrated tool registry
        this.tier3 = new UnifiedExecutor(
            executorConfig,
            this.eventBus!,
            this.logger,
            this.toolRegistry!,
        );

        this.logger.debug("[ExecutionArchitecture] Tier 3 initialized");
    }

    /**
     * Initialize Tier 2 - Process Intelligence
     */
    private async initializeTier2(): Promise<void> {
        this.logger.debug("[ExecutionArchitecture] Initializing Tier 2");

        if (!this.tier3) {
            throw new Error("Tier 3 must be initialized before Tier 2");
        }

        // Create Tier 2 with dependency on Tier 3 and SwarmContextManager
        this.tier2 = new TierTwoOrchestrator(
            this.logger,
            this.eventBus!,
            this.tier3,
            this.swarmContextManager || undefined, // Pass SwarmContextManager if available
        );

        this.logger.debug("[ExecutionArchitecture] Tier 2 initialized");
    }

    /**
     * Initialize Tier 1 - Coordination Intelligence
     */
    private async initializeTier1(): Promise<void> {
        this.logger.debug("[ExecutionArchitecture] Initializing Tier 1");

        if (!this.tier2) {
            throw new Error("Tier 2 must be initialized before Tier 1");
        }

        if (!this.swarmContextManager) {
            throw new Error("SwarmContextManager must be initialized for Tier 1");
        }

        // Create conversation bridge for AI coordination
        const conversationBridge = createConversationBridge(this.logger);

        // Create Tier 1 with SwarmCoordinator (direct implementation)
        this.tier1 = new SwarmCoordinator(
            this.logger,
            this.swarmContextManager,
            conversationBridge,
            this.tier2,
        );

        this.logger.debug("[ExecutionArchitecture] Tier 1 initialized with SwarmCoordinator", {
            hasSwarmContextManager: true,
            architecture: "direct-coordination",
        });
    }

    /**
     * Wire up event listeners for cross-tier communication
     */
    private async wireEventListeners(): Promise<void> {
        this.logger.debug("[ExecutionArchitecture] Wiring event listeners");

        if (!this.eventBus) {
            throw new Error("EventBus not initialized");
        }

        // Subscribe to cross-tier events
        await this.eventBus.subscribe({
            id: generatePK().toString(),
            pattern: "tier.*",
            handler: async (event) => {
                this.logger.debug("[ExecutionArchitecture] Cross-tier event received", {
                    type: event.type,
                    source: event.source,
                });
            },
            filters: [
                { field: "source.tier", operator: "in", value: ["tier1", "tier2", "tier3"] },
            ],
        });

        // Subscribe to telemetry events for monitoring
        // Telemetry subscription removed - monitoring now handled by emergent agents

        // History pattern detection subscription removed - monitoring now handled by emergent agents

        this.logger.debug("[ExecutionArchitecture] Event listeners wired");
    }

    /**
     * Start all components
     */
    async start(): Promise<void> {
        if (!this.initialized) {
            throw new Error("ExecutionArchitecture not initialized. Call create() first.");
        }

        this.logger.info("[ExecutionArchitecture] Starting execution architecture");

        // Components are already running after initialization
        // This method is for future extensions where components might need explicit start

        this.logger.info("[ExecutionArchitecture] Execution architecture started");
    }

    /**
     * Stop all components gracefully
     */
    async stop(): Promise<void> {
        this.logger.info("[ExecutionArchitecture] Stopping execution architecture");

        try {
            // Resource monitor functionality now provided by emergent agents
            // if (this.resourceMonitor) {
            //     this.resourceMonitor.shutdown();
            // }

            // Shutdown modern state management components
            if (this.swarmContextManager) {
                await this.swarmContextManager.shutdown();
            }

            if (this.contextSubscriptionManager) {
                await this.contextSubscriptionManager.shutdown();
            }

            // Stop event bus
            if (this.eventBus) {
                await this.eventBus.stop();
            }

            // Cleanup state stores if they have cleanup methods
            // (Redis connections are handled by the stores themselves)

            this.logger.info("[ExecutionArchitecture] Execution architecture stopped");

        } catch (error) {
            this.logger.error("[ExecutionArchitecture] Error during shutdown", {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get Tier 1 - Coordination Intelligence
     */
    getTier1(): TierCommunicationInterface {
        if (!this.tier1) {
            throw new Error("Tier 1 not initialized");
        }
        return this.tier1;
    }

    /**
     * Get Tier 2 - Process Intelligence
     */
    getTier2(): TierCommunicationInterface {
        if (!this.tier2) {
            throw new Error("Tier 2 not initialized");
        }
        return this.tier2;
    }

    /**
     * Get Tier 3 - Execution Intelligence
     */
    getTier3(): TierCommunicationInterface {
        if (!this.tier3) {
            throw new Error("Tier 3 not initialized");
        }
        return this.tier3;
    }

    /**
     * Get Integrated Tool Registry
     */
    getToolRegistry(): IntegratedToolRegistry {
        if (!this.toolRegistry) {
            throw new Error("Tool registry not initialized");
        }
        return this.toolRegistry;
    }

    /**
     * Get SwarmContextManager (if using modern state management)
     */
    getSwarmContextManager(): ISwarmContextManager | null {
        return this.swarmContextManager;
    }

    /**
     * Get ContextSubscriptionManager (if using modern state management)
     */
    getContextSubscriptionManager(): ContextSubscriptionManager | null {
        return this.contextSubscriptionManager;
    }

    /**
     * Get combined capabilities of all tiers
     */
    async getCapabilities(): Promise<Record<string, TierCapabilities>> {
        const capabilities: Record<string, TierCapabilities> = {};

        if (this.tier1) {
            capabilities.tier1 = await this.tier1.getCapabilities();
        }

        if (this.tier2) {
            capabilities.tier2 = await this.tier2.getCapabilities();
        }

        if (this.tier3) {
            capabilities.tier3 = await this.tier3.getCapabilities();
        }

        return capabilities;
    }

    /**
     * Get current architecture status
     */
    getStatus(): {
        initialized: boolean;
        tier1Ready: boolean;
        tier2Ready: boolean;
        tier3Ready: boolean;
        eventBusReady: boolean;
        runStateStoreReady: boolean;
        toolRegistryReady: boolean;
        modernStateManagement: boolean;
        swarmContextManagerReady: boolean;
        contextSubscriptionManagerReady: boolean;
        // resourceManagerReady removed - monitoring now emergent
        monitoringToolsReady: boolean;
    } {
        return {
            initialized: this.initialized,
            tier1Ready: !!this.tier1,
            tier2Ready: !!this.tier2,
            tier3Ready: !!this.tier3,
            eventBusReady: !!this.eventBus,
            runStateStoreReady: !!this.runStateStore,
            toolRegistryReady: !!this.toolRegistry,
            modernStateManagement: !!this.swarmContextManager,
            swarmContextManagerReady: !!this.swarmContextManager,
            contextSubscriptionManagerReady: !!this.contextSubscriptionManager,
            // resourceManagerReady removed - monitoring now emergent
            monitoringToolsReady: !!this.toolRegistry && !!(this.toolRegistry as any)._monitoringToolInstances,
        };
    }

    /**
     * Clean up any partially initialized components
     */
    private async cleanup(): Promise<void> {
        this.logger.debug("[ExecutionArchitecture] Cleaning up components");

        // Reset tier references
        this.tier1 = null;
        this.tier2 = null;
        this.tier3 = null;

        // Cleanup modern state management components
        if (this.swarmContextManager) {
            try {
                await this.swarmContextManager.shutdown();
            } catch (error) {
                this.logger.error("[ExecutionArchitecture] Error stopping SwarmContextManager", {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
            this.swarmContextManager = null;
        }

        if (this.contextSubscriptionManager) {
            try {
                await this.contextSubscriptionManager.shutdown();
            } catch (error) {
                this.logger.error("[ExecutionArchitecture] Error stopping ContextSubscriptionManager", {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
            this.contextSubscriptionManager = null;
        }

        // Stop event bus if initialized
        if (this.eventBus) {
            try {
                await this.eventBus.stop();
            } catch (error) {
                this.logger.error("[ExecutionArchitecture] Error stopping event bus", {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
            this.eventBus = null;
        }

        // Clear state stores
        this.runStateStore = null;

        // Rolling history cleanup removed - monitoring now handled by emergent agents

        // Clear tool registry and conversation store
        this.toolRegistry = null;
        this.conversationStore = null;

        // Clear resource managers
        this.tier1ResourceManager = null;
        this.tier3ResourceManager = null;

        this.initialized = false;
    }
}

/**
 * Singleton instance management
 */
let architectureInstance: ExecutionArchitecture | null = null;

/**
 * Get or create the execution architecture instance
 * This provides a convenient singleton pattern for the architecture
 */
export async function getExecutionArchitecture(
    options?: ExecutionArchitectureOptions,
): Promise<ExecutionArchitecture> {
    if (!architectureInstance) {
        architectureInstance = await ExecutionArchitecture.create(options || {});
    }
    return architectureInstance;
}

/**
 * Reset the singleton instance (mainly for testing)
 */
export async function resetExecutionArchitecture(): Promise<void> {
    if (architectureInstance) {
        await architectureInstance.stop();
        architectureInstance = null;
    }
}
