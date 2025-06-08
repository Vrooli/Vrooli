import { StrategyType } from '@vrooli/shared';
import { expect } from 'chai';
import sinon from 'sinon';
import { createMockLogger } from '../../../../__test/fixtures/loggerFixtures.js';
import { EventBus } from '../../../cross-cutting/events/eventBus.js';
import { ToolOrchestrator } from '../../engine/toolOrchestrator.js';
import { ValidationEngine } from '../../engine/validationEngine.js';
import { ConversationalStrategy } from '../conversationalStrategy.js';
import { DeterministicStrategy } from '../deterministicStrategy.js';
import { StrategyFactory } from '../strategyFactory.js';

describe('Tool Integration in Strategies (Mocha)', function () {
    let logger: any;
    let sandbox: sinon.SinonSandbox;

    beforeEach(function () {
        sandbox = sinon.createSandbox();
        logger = createMockLogger(sandbox);
    });

    afterEach(function () {
        sandbox.restore();
    });

    describe('Dependency Injection', function () {
        it('should create ConversationalStrategy with shared services', function () {
            const eventBus = new EventBus(logger);
            const toolOrchestrator = new ToolOrchestrator(eventBus, logger);
            const validationEngine = new ValidationEngine(logger);

            // Create strategy with shared services - should not throw
            expect(() => {
                new ConversationalStrategy(logger, toolOrchestrator, validationEngine);
            }).to.not.throw();
        });

        it('should create DeterministicStrategy with shared services', function () {
            const eventBus = new EventBus(logger);
            const toolOrchestrator = new ToolOrchestrator(eventBus, logger);
            const validationEngine = new ValidationEngine(logger);

            // Create strategy with shared services - should not throw
            expect(() => {
                new DeterministicStrategy(logger, toolOrchestrator, validationEngine);
            }).to.not.throw();
        });

        it('should allow setting services via setter methods', function () {
            const eventBus = new EventBus(logger);
            const toolOrchestrator = new ToolOrchestrator(eventBus, logger);
            const validationEngine = new ValidationEngine(logger);

            // Create strategies without services
            const conversationalStrategy = new ConversationalStrategy(logger);
            const deterministicStrategy = new DeterministicStrategy(logger);

            // Set services via setters - should not throw
            expect(() => {
                conversationalStrategy.setToolOrchestrator(toolOrchestrator);
                conversationalStrategy.setValidationEngine(validationEngine);
                deterministicStrategy.setToolOrchestrator(toolOrchestrator);
                deterministicStrategy.setValidationEngine(validationEngine);
            }).to.not.throw();
        });
    });

    describe('StrategyFactory Integration', function () {
        it('should create factory with shared services', function () {
            const eventBus = new EventBus(logger);
            const toolOrchestrator = new ToolOrchestrator(eventBus, logger);
            const validationEngine = new ValidationEngine(logger);

            // Create factory with shared services - should not throw
            expect(() => {
                new StrategyFactory(logger, toolOrchestrator, validationEngine);
            }).to.not.throw();
        });

        it('should inject services into strategies created by factory', function () {
            const eventBus = new EventBus(logger);
            const toolOrchestrator = new ToolOrchestrator(eventBus, logger);
            const validationEngine = new ValidationEngine(logger);

            const factory = new StrategyFactory(logger, toolOrchestrator, validationEngine);

            // Get strategies from factory
            const conversationalStrategy = factory.getStrategy(StrategyType.CONVERSATIONAL);
            const deterministicStrategy = factory.getStrategy(StrategyType.DETERMINISTIC);

            // Verify strategies were created
            expect(conversationalStrategy).to.exist;
            expect(deterministicStrategy).to.exist;
            expect(conversationalStrategy?.type).to.equal(StrategyType.CONVERSATIONAL);
            expect(deterministicStrategy?.type).to.equal(StrategyType.DETERMINISTIC);
        });

        it('should update existing strategies when services are set', function () {
            const eventBus = new EventBus(logger);
            const toolOrchestrator = new ToolOrchestrator(eventBus, logger);
            const validationEngine = new ValidationEngine(logger);

            // Create factory without services
            const factory = new StrategyFactory(logger);

            // Get a strategy
            const strategy = factory.getStrategy(StrategyType.CONVERSATIONAL);
            expect(strategy).to.exist;

            // Set services on factory
            factory.setSharedServices(toolOrchestrator, validationEngine);

            // Verify no errors thrown
            expect(factory.getStrategy(StrategyType.CONVERSATIONAL)).to.exist;
        });
    });
});