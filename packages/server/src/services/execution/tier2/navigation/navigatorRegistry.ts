import { type Logger } from "winston";
import { type Navigator } from "@vrooli/shared";
import { NativeNavigator } from "./navigators/nativeNavigator.js";
// import { BPMNNavigator } from "./navigators/bpmnNavigator.js";
// import { LangchainNavigator } from "./navigators/langchainNavigator.js";
// import { TemporalNavigator } from "./navigators/temporalNavigator.js";

/**
 * NavigatorRegistry - Universal navigator management
 * 
 * This registry manages different navigator implementations that enable
 * Vrooli to execute workflows from various platforms. Each navigator
 * translates platform-specific workflow definitions into a common
 * navigation interface.
 * 
 * Supported platforms:
 * - Native Vrooli workflows
 * - BPMN (Business Process Model and Notation)
 * - Langchain (AI chain orchestration)
 * - Temporal (distributed workflow engine)
 * - Custom formats via plugin system
 * 
 * This creates a universal automation ecosystem where workflows from
 * different platforms can interoperate seamlessly.
 */
export class NavigatorRegistry {
    private readonly navigators: Map<string, Navigator> = new Map();
    private readonly logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
        this.initializeDefaultNavigators();
    }

    /**
     * Initializes built-in navigators
     */
    private initializeDefaultNavigators(): void {
        // Register native Vrooli navigator
        this.registerNavigator(new NativeNavigator(this.logger));

        // Register other platform navigators
        // this.registerNavigator(new BPMNNavigator(this.logger));
        // this.registerNavigator(new LangchainNavigator(this.logger));
        // this.registerNavigator(new TemporalNavigator(this.logger));

        this.logger.info("[NavigatorRegistry] Default navigators initialized", {
            navigatorTypes: Array.from(this.navigators.keys()),
        });
    }

    /**
     * Registers a navigator implementation
     */
    registerNavigator(navigator: Navigator): void {
        const type = navigator.type;
        
        if (this.navigators.has(type)) {
            this.logger.warn(`[NavigatorRegistry] Overwriting existing navigator: ${type}`);
        }

        this.navigators.set(type, navigator);
        
        this.logger.info("[NavigatorRegistry] Registered navigator", {
            type,
            version: navigator.version,
        });
    }

    /**
     * Gets a navigator by type
     */
    getNavigator(type: string): Navigator {
        const navigator = this.navigators.get(type);
        
        if (!navigator) {
            throw new Error(`Navigator not found for type: ${type}`);
        }

        return navigator;
    }

    /**
     * Checks if a navigator type is available
     */
    hasNavigator(type: string): boolean {
        return this.navigators.has(type);
    }

    /**
     * Lists all registered navigator types
     */
    listNavigatorTypes(): string[] {
        return Array.from(this.navigators.keys());
    }

    /**
     * Gets navigator info
     */
    getNavigatorInfo(type: string): { type: string; version: string } | null {
        const navigator = this.navigators.get(type);
        
        if (!navigator) {
            return null;
        }

        return {
            type: navigator.type,
            version: navigator.version,
        };
    }

    /**
     * Unregisters a navigator
     */
    unregisterNavigator(type: string): boolean {
        const removed = this.navigators.delete(type);
        
        if (removed) {
            this.logger.info(`[NavigatorRegistry] Unregistered navigator: ${type}`);
        }

        return removed;
    }

    /**
     * Auto-detects the appropriate navigator for a routine
     */
    detectNavigatorType(routineDefinition: unknown): string | null {
        // Try each navigator's canNavigate method
        for (const [type, navigator] of this.navigators) {
            try {
                if (navigator.canNavigate(routineDefinition)) {
                    this.logger.debug(`[NavigatorRegistry] Auto-detected navigator type: ${type}`);
                    return type;
                }
            } catch (error) {
                this.logger.debug(`[NavigatorRegistry] Navigator ${type} cannot handle definition`, {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        return null;
    }

    /**
     * Validates a routine definition with a specific navigator
     */
    validateRoutineDefinition(
        type: string,
        definition: unknown,
    ): { valid: boolean; errors?: string[] } {
        const navigator = this.navigators.get(type);
        
        if (!navigator) {
            return {
                valid: false,
                errors: [`Navigator not found for type: ${type}`],
            };
        }

        try {
            const canNavigate = navigator.canNavigate(definition);
            
            if (!canNavigate) {
                return {
                    valid: false,
                    errors: ["Navigator cannot handle this definition format"],
                };
            }

            // Try to get start location as additional validation
            navigator.getStartLocation(definition);

            return { valid: true };

        } catch (error) {
            return {
                valid: false,
                errors: [error instanceof Error ? error.message : "Invalid definition"],
            };
        }
    }

    /**
     * Gets metrics for all navigators
     */
    getNavigatorMetrics(): Record<string, NavigatorMetrics> {
        const metrics: Record<string, NavigatorMetrics> = {};

        for (const [type, navigator] of this.navigators) {
            // TODO: Implement actual metrics collection
            metrics[type] = {
                type,
                version: navigator.version,
                routinesProcessed: 0,
                averageStepTime: 0,
                successRate: 1.0,
            };
        }

        return metrics;
    }
}

/**
 * Navigator metrics
 */
interface NavigatorMetrics {
    type: string;
    version: string;
    routinesProcessed: number;
    averageStepTime: number;
    successRate: number;
}
