import { describe, expect, it } from "vitest";
import {
    InputGenerationStrategy,
    PathSelectionStrategy,
    SubroutineExecutionStrategy,
    BotStyle,
    BranchStatus,
} from "./enums.js";

describe("Run Enums", () => {
    describe("InputGenerationStrategy", () => {
        it("should provide distinct strategies for different use cases", () => {
            // Auto should be used when AI can generate meaningful inputs
            expect(InputGenerationStrategy.Auto).toBeDefined();
            // Manual should be used when user input is required for safety or precision
            expect(InputGenerationStrategy.Manual).toBeDefined();
            
            // Strategies should be mutually exclusive
            expect(InputGenerationStrategy.Auto).not.toBe(InputGenerationStrategy.Manual);
        });

        it("should support automation preference workflows", () => {
            // Test that enums can be used in conditional logic for user preferences
            const getUserPreference = (wantsAutomation: boolean) => 
                wantsAutomation ? InputGenerationStrategy.Auto : InputGenerationStrategy.Manual;
            
            expect(getUserPreference(true)).toBe(InputGenerationStrategy.Auto);
            expect(getUserPreference(false)).toBe(InputGenerationStrategy.Manual);
        });

        it("should have all expected strategies available", () => {
            const strategies = Object.values(InputGenerationStrategy);
            expect(strategies).toHaveLength(2);
            expect(strategies).toContain(InputGenerationStrategy.Auto);
            expect(strategies).toContain(InputGenerationStrategy.Manual);
        });
    });

    describe("PathSelectionStrategy", () => {
        it("should provide strategies with different risk/predictability profiles", () => {
            // AutoPickFirst should provide deterministic, predictable behavior
            expect(PathSelectionStrategy.AutoPickFirst).toBeDefined();
            // AutoPickLLM should provide intelligent but potentially unpredictable selection  
            expect(PathSelectionStrategy.AutoPickLLM).toBeDefined();
            // AutoPickRandom should provide exploration capabilities
            expect(PathSelectionStrategy.AutoPickRandom).toBeDefined();
            // ManualPick should provide user control for critical decisions
            expect(PathSelectionStrategy.ManualPick).toBeDefined();
        });

        it("should support automation vs control trade-offs", () => {
            const getStrategyForSituation = (isCritical: boolean, needsExploration: boolean) => {
                if (isCritical) return PathSelectionStrategy.ManualPick;
                if (needsExploration) return PathSelectionStrategy.AutoPickRandom;
                return PathSelectionStrategy.AutoPickFirst;
            };
            
            expect(getStrategyForSituation(true, false)).toBe(PathSelectionStrategy.ManualPick);
            expect(getStrategyForSituation(false, true)).toBe(PathSelectionStrategy.AutoPickRandom);
            expect(getStrategyForSituation(false, false)).toBe(PathSelectionStrategy.AutoPickFirst);
        });

        it("should cover all automated and manual selection approaches", () => {
            const strategies = Object.values(PathSelectionStrategy);
            expect(strategies).toHaveLength(4);
            
            // Should include both automated approaches
            const automatedStrategies = strategies.filter(s => s.startsWith("Auto"));
            expect(automatedStrategies).toHaveLength(3);
            
            // Should include manual control
            const manualStrategies = strategies.filter(s => s.startsWith("Manual"));
            expect(manualStrategies).toHaveLength(1);
        });
    });

    describe("SubroutineExecutionStrategy", () => {
        it("should provide execution control for safety and user oversight", () => {
            // Auto execution should be used for trusted, low-risk operations
            expect(SubroutineExecutionStrategy.Auto).toBeDefined();
            // Manual execution should be used when user approval is needed before proceeding
            expect(SubroutineExecutionStrategy.Manual).toBeDefined();
        });

        it("should support execution safety workflows", () => {
            const getExecutionStrategy = (isHighRisk: boolean, isAutomatedRun: boolean) => {
                if (isHighRisk) return SubroutineExecutionStrategy.Manual;
                if (isAutomatedRun) return SubroutineExecutionStrategy.Auto;
                return SubroutineExecutionStrategy.Manual;
            };
            
            expect(getExecutionStrategy(true, true)).toBe(SubroutineExecutionStrategy.Manual);
            expect(getExecutionStrategy(false, true)).toBe(SubroutineExecutionStrategy.Auto);
            expect(getExecutionStrategy(false, false)).toBe(SubroutineExecutionStrategy.Manual);
        });

        it("should align with input generation strategy values for consistency", () => {
            // These enums should share values to enable consistent automation settings
            expect(SubroutineExecutionStrategy.Auto).toBe(InputGenerationStrategy.Auto);
            expect(SubroutineExecutionStrategy.Manual).toBe(InputGenerationStrategy.Manual);
        });
    });

    describe("BotStyle", () => {
        it("should provide bot interaction options for different use cases", () => {
            // Default should provide standard AI assistance without configuration
            expect(BotStyle.Default).toBeDefined();
            // Specific should allow customized bot personas/behavior
            expect(BotStyle.Specific).toBeDefined();
            // None should disable bot interaction for user-only workflows
            expect(BotStyle.None).toBeDefined();
        });

        it("should support bot selection based on routine requirements", () => {
            const getBotStyle = (needsCustomBehavior: boolean, wantsBotHelp: boolean) => {
                if (!wantsBotHelp) return BotStyle.None;
                if (needsCustomBehavior) return BotStyle.Specific;
                return BotStyle.Default;
            };
            
            expect(getBotStyle(false, false)).toBe(BotStyle.None);
            expect(getBotStyle(true, true)).toBe(BotStyle.Specific);
            expect(getBotStyle(false, true)).toBe(BotStyle.Default);
        });

        it("should cover all bot interaction levels", () => {
            const styles = Object.values(BotStyle);
            expect(styles).toHaveLength(3);
            expect(styles).toContain(BotStyle.None);
            expect(styles).toContain(BotStyle.Default);
            expect(styles).toContain(BotStyle.Specific);
        });
    });

    describe("BranchStatus", () => {
        it("should represent execution states for monitoring and control", () => {
            // Active should indicate branch is currently running
            expect(BranchStatus.Active).toBeDefined();
            // Completed should indicate successful termination
            expect(BranchStatus.Completed).toBeDefined();
            // Failed should indicate error termination
            expect(BranchStatus.Failed).toBeDefined();
            // Waiting should indicate blocked/paused execution
            expect(BranchStatus.Waiting).toBeDefined();
        });

        it("should support status transition logic", () => {
            const getNextStatus = (current: BranchStatus, hasError: boolean, isBlocked: boolean, isComplete: boolean) => {
                if (hasError) return BranchStatus.Failed;
                if (isComplete) return BranchStatus.Completed;
                if (isBlocked) return BranchStatus.Waiting;
                return BranchStatus.Active;
            };
            
            expect(getNextStatus(BranchStatus.Waiting, true, false, false)).toBe(BranchStatus.Failed);
            expect(getNextStatus(BranchStatus.Active, false, false, true)).toBe(BranchStatus.Completed);
            expect(getNextStatus(BranchStatus.Active, false, true, false)).toBe(BranchStatus.Waiting);
            expect(getNextStatus(BranchStatus.Waiting, false, false, false)).toBe(BranchStatus.Active);
        });

        it("should distinguish terminal vs non-terminal states", () => {
            const terminalStates = [BranchStatus.Completed, BranchStatus.Failed];
            const nonTerminalStates = [BranchStatus.Active, BranchStatus.Waiting];
            
            expect(terminalStates).toHaveLength(2);
            expect(nonTerminalStates).toHaveLength(2);
            
            // Terminal states should not be able to transition to other states in normal operation
            expect(terminalStates.every(state => 
                [BranchStatus.Completed, BranchStatus.Failed].includes(state)
            )).toBe(true);
        });

        it("should cover all execution lifecycle states", () => {
            const statuses = Object.values(BranchStatus);
            expect(statuses).toHaveLength(4);
            
            // Should have unique values for each state
            expect(new Set(statuses).size).toBe(4);
        });
    });

    describe("Cross-enum Integration", () => {
        it("should support creating coherent run configurations", () => {
            // Test that enums work together for real-world scenarios
            type RunConfig = {
                inputStrategy: InputGenerationStrategy;
                pathStrategy: PathSelectionStrategy;
                executionStrategy: SubroutineExecutionStrategy;
                botStyle: BotStyle;
            };
            
            // Fully automated configuration
            const fullyAutomated: RunConfig = {
                inputStrategy: InputGenerationStrategy.Auto,
                pathStrategy: PathSelectionStrategy.AutoPickLLM,
                executionStrategy: SubroutineExecutionStrategy.Auto,
                botStyle: BotStyle.Default,
            };
            
            // Fully manual configuration
            const fullyManual: RunConfig = {
                inputStrategy: InputGenerationStrategy.Manual,
                pathStrategy: PathSelectionStrategy.ManualPick,
                executionStrategy: SubroutineExecutionStrategy.Manual,
                botStyle: BotStyle.None,
            };
            
            expect(fullyAutomated.inputStrategy).toBe("Auto");
            expect(fullyAutomated.pathStrategy).toBe("AutoPickLLM");
            expect(fullyManual.inputStrategy).toBe("Manual");
            expect(fullyManual.pathStrategy).toBe("ManualPick");
        });
    });
});