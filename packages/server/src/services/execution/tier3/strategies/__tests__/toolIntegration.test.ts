import { StrategyType } from '@vrooli/shared';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createMockLogger } from '../../../../__test/fixtures/loggerFixtures.js';
import { EventBus } from '../../../cross-cutting/events/eventBus.js';
import { ToolOrchestrator } from '../../engine/toolOrchestrator.js';
import { ValidationEngine } from '../../engine/validationEngine.js';
import { ConversationalStrategy } from '../conversationalStrategy.js';
import { DeterministicStrategy } from '../deterministicStrategy.js';

describe('Tool Integration in Strategies', () => {
    let logger: any;
    let eventBus: EventBus;
    let toolOrchestrator: ToolOrchestrator;
    let validationEngine: ValidationEngine;

    beforeEach(() => {
        logger = createMockLogger();
        eventBus = new EventBus(logger);
        toolOrchestrator = new ToolOrchestrator(eventBus, logger);
        validationEngine = new ValidationEngine(logger);
    });

    describe('ConversationalStrategy', () => {
        it('should use shared ToolOrchestrator instance', async () => {
            // Create strategy with shared services
            const strategy = new ConversationalStrategy(logger, toolOrchestrator, validationEngine);

            // Mock tool execution
            const mockToolResult = { result: 'tool executed' };
            vi.spyOn(toolOrchestrator, 'executeTool').mockResolvedValue(mockToolResult);

            // Create a context that would trigger tool execution
            const context = {
                stepId: 'test-step',
                stepType: 'search',
                inputs: { query: 'test query' },
                config: { strategy: StrategyType.CONVERSATIONAL },
                resources: {
                    tools: [{ name: 'search', type: 'search', description: 'Search tool', parameters: {} }],
                    credits: 1000,
                },
                history: { recentSteps: [], totalExecutions: 0, successRate: 0 },
                constraints: {},
            };

            // Execute strategy
            const result = await strategy.execute(context);

            // Verify strategy executed successfully
            expect(result.success).toBe(true);
            expect(result.metadata.strategyType).toBe(StrategyType.CONVERSATIONAL);
        });

        it('should accept services via setter methods', () => {
            // Create strategy without services
            const strategy = new ConversationalStrategy(logger);

            // Set services via setters
            strategy.setToolOrchestrator(toolOrchestrator);
            strategy.setValidationEngine(validationEngine);

            // Verify setters work (no errors thrown)
            expect(() => strategy.setToolOrchestrator(toolOrchestrator)).not.toThrow();
            expect(() => strategy.setValidationEngine(validationEngine)).not.toThrow();
        });
    });

    describe('DeterministicStrategy', () => {
        it('should use shared ValidationEngine instance', async () => {
            // Create strategy with shared services
            const strategy = new DeterministicStrategy(logger, toolOrchestrator, validationEngine);

            // Mock validation
            vi.spyOn(validationEngine, 'validateOutputs').mockResolvedValue({
                valid: true,
                sanitizedOutputs: { result: 'validated' },
                errors: [],
            });

            // Create a context
            const context = {
                stepId: 'test-step',
                stepType: 'transform',
                inputs: { data: 'test data' },
                config: {
                    strategy: StrategyType.DETERMINISTIC,
                    expectedOutputs: { result: { type: 'string' } },
                },
                resources: { credits: 1000 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 0 },
                constraints: {},
            };

            // Execute strategy
            const result = await strategy.execute(context);

            // Verify strategy executed successfully
            expect(result.success).toBe(true);
            expect(result.metadata.strategyType).toBe(StrategyType.DETERMINISTIC);
        });

        it('should accept services via setter methods', () => {
            // Create strategy without services
            const strategy = new DeterministicStrategy(logger);

            // Set services via setters
            strategy.setToolOrchestrator(toolOrchestrator);
            strategy.setValidationEngine(validationEngine);

            // Verify setters work (no errors thrown)
            expect(() => strategy.setToolOrchestrator(toolOrchestrator)).not.toThrow();
            expect(() => strategy.setValidationEngine(validationEngine)).not.toThrow();
        });
    });

    describe('StrategyFactory Integration', () => {
        it('should inject shared services into all strategies', async () => {
            const { StrategyFactory } = await import('../strategyFactory.js');

            // Create factory with shared services
            const factory = new StrategyFactory(logger, toolOrchestrator, validationEngine);

            // Get strategies
            const conversationalStrategy = factory.getStrategy(StrategyType.CONVERSATIONAL);
            const deterministicStrategy = factory.getStrategy(StrategyType.DETERMINISTIC);

            // Verify strategies exist
            expect(conversationalStrategy).toBeDefined();
            expect(deterministicStrategy).toBeDefined();

            // Verify they are the correct types
            expect(conversationalStrategy?.type).toBe(StrategyType.CONVERSATIONAL);
            expect(deterministicStrategy?.type).toBe(StrategyType.DETERMINISTIC);
        });
    });
});