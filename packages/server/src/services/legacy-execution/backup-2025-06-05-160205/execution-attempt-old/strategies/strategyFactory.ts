import { type ResourceVersion } from "@vrooli/shared";
import { type Logger } from "winston";
import { logger as defaultLogger } from "../../../events/logger.js";
import { ConversationalStrategy } from "./conversationalStrategy.js";
import { DeterministicStrategy } from "./deterministicStrategy.js";
import { type ExecutionStrategy } from "./executionStrategy.js";
import { ReasoningStrategy } from "./reasoningStrategy.js";

/**
 * Factory for creating execution strategies based on routine configuration.
 * 
 * The factory uses a priority-based selection process:
 * 1. Explicit strategy configuration (executionStrategy field)
 * 2. Routine subtype matching
 * 3. Keyword-based heuristics
 * 4. Default fallback to reasoning strategy
 */
export class StrategyFactory {
    private readonly strategies: ExecutionStrategy[];
    private readonly logger: Logger;

    constructor(logger?: Logger) {
        this.logger = logger || defaultLogger;

        // Initialize all available strategies
        this.strategies = [
            new DeterministicStrategy(),
            new ReasoningStrategy(),
            new ConversationalStrategy(),
        ];
    }

    /**
     * Selects the most appropriate execution strategy for a routine.
     * 
     * @param routine The routine to execute
     * @returns The selected execution strategy
     */
    selectStrategy(routine: ResourceVersion): ExecutionStrategy {
        const routineSubType = routine.resourceSubType || "";
        const config = routine.config || {};
        const explicitStrategy = (config as any).executionStrategy;

        this.logger.debug(`Selecting strategy for routine ${routine.id}`, {
            subType: routineSubType,
            hasConfig: !!config,
            configStrategy: explicitStrategy,
        });

        // Check each strategy in priority order
        for (const strategy of this.strategies) {
            if (strategy.canHandle(routineSubType, config)) {
                this.logger.info(`Selected ${strategy.name} for routine ${routine.id}`);
                return strategy;
            }
        }

        // Default fallback to reasoning strategy
        const defaultStrategy = this.strategies.find(s => s.name === "ReasoningStrategy")!;
        this.logger.info(`Using default ${defaultStrategy.name} for routine ${routine.id}`);
        return defaultStrategy;
    }

    /**
     * Gets a strategy by name.
     * 
     * @param name The name of the strategy
     * @returns The strategy if found, null otherwise
     */
    getStrategyByName(name: string): ExecutionStrategy | null {
        return this.strategies.find(s => s.name === name) || null;
    }

    /**
     * Lists all available strategies.
     * 
     * @returns Array of strategy names
     */
    listAvailableStrategies(): string[] {
        return this.strategies.map(s => s.name);
    }

    /**
     * Analyzes a routine and returns information about which strategy would be selected and why.
     * Useful for debugging and testing.
     * 
     * @param routine The routine to analyze
     * @returns Analysis results
     */
    analyzeRoutine(routine: ResourceVersion): {
        selectedStrategy: string;
        reason: string;
        alternativeStrategies: string[];
        routineInfo: {
            subType: string;
            hasConfig: boolean;
            explicitStrategy?: string;
        };
    } {
        const routineSubType = routine.resourceSubType || "";
        const config = routine.config || {};
        const explicitStrategy = (config as any).executionStrategy;
        const selectedStrategy = this.selectStrategy(routine);

        // Determine why this strategy was selected
        let reason = "Default fallback";

        if (explicitStrategy) {
            reason = `Explicit configuration: executionStrategy = "${explicitStrategy}"`;
        } else if (selectedStrategy.canHandle(routineSubType, {})) {
            // Strategy can handle based on subtype alone
            reason = `Routine subtype "${routineSubType}" is handled by ${selectedStrategy.name}`;
        } else {
            // Must be keyword-based
            reason = `Keywords in routine name/description matched ${selectedStrategy.name}`;
        }

        // Find alternative strategies that could handle this routine
        const alternativeStrategies = this.strategies
            .filter(s => s !== selectedStrategy && s.canHandle(routineSubType, config))
            .map(s => s.name);

        return {
            selectedStrategy: selectedStrategy.name,
            reason,
            alternativeStrategies,
            routineInfo: {
                subType: routineSubType,
                hasConfig: !!config,
                explicitStrategy: explicitStrategy as string | undefined,
            },
        };
    }

    /**
     * Registers a custom strategy.
     * Useful for extending the factory with domain-specific strategies.
     * 
     * @param strategy The strategy to register
     */
    registerStrategy(strategy: ExecutionStrategy): void {
        // Check if strategy with same name already exists
        const existingIndex = this.strategies.findIndex(s => s.name === strategy.name);

        if (existingIndex >= 0) {
            this.logger.warn(`Replacing existing strategy "${strategy.name}"`);
            this.strategies[existingIndex] = strategy;
        } else {
            this.logger.info(`Registering new strategy "${strategy.name}"`);
            // Add new strategies at the beginning for higher priority
            this.strategies.unshift(strategy);
        }
    }

    /**
     * Unregisters a strategy by name.
     * 
     * @param name The name of the strategy to unregister
     * @returns True if the strategy was found and removed
     */
    unregisterStrategy(name: string): boolean {
        const index = this.strategies.findIndex(s => s.name === name);

        if (index >= 0) {
            // Don't allow removing all strategies
            if (this.strategies.length === 1) {
                this.logger.error("Cannot unregister the last remaining strategy");
                return false;
            }

            this.strategies.splice(index, 1);
            this.logger.info(`Unregistered strategy "${name}"`);
            return true;
        }

        this.logger.warn(`Strategy "${name}" not found for unregistration`);
        return false;
    }
} 
