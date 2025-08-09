/**
 * BotPriority service tests
 * 
 * Tests for bot priority calculation and decision making logic.
 * This is critical infrastructure for bot selection and event handling.
 */

import type { BotConfigObject } from "@vrooli/shared";
import { generatePK } from "@vrooli/shared";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { BotPriorityCalculator, DefaultDecisionMaker } from "./BotPriority.js";
import { PATTERN_SCORING, PRIORITY_WEIGHTS, ROLE_WEIGHTS } from "./constants.js";
import type { BotDecisionContext, BotParticipant, ServiceEvent } from "./types.js";

// Mock logger to suppress output during tests
vi.mock("../../events/logger.js", () => ({
    logger: {
        debug: vi.fn(),
        error: vi.fn(),
        info: vi.fn(),
        warning: vi.fn(),
    },
}));

// Mock registry for event behavior
vi.mock("./registry.js", () => ({
    getEventBehavior: vi.fn().mockReturnValue({
        interceptable: true,
        mode: "interceptable",
    }),
}));

describe("BotPriority", () => {
    let calculator: BotPriorityCalculator;
    let decisionMaker: DefaultDecisionMaker;

    beforeAll(() => {
        calculator = new BotPriorityCalculator();
        decisionMaker = new DefaultDecisionMaker();
    });

    // Helper function to create test bot participants
    function createTestBot(overrides: Partial<BotParticipant> = {}): BotParticipant {
        return {
            id: generatePK().toString(),
            name: "Test Bot",
            state: "ready",
            config: {
                __version: "1.0",
                resources: [],
                agentSpec: {
                    behaviors: [],
                },
            } as BotConfigObject,
            ...overrides,
        };
    }

    // Helper function to create test events
    function createTestEvent(overrides: Partial<ServiceEvent> = {}): ServiceEvent {
        return {
            id: generatePK().toString(),
            type: "finance/transaction/completed",
            timestamp: new Date(),
            data: { test: "data" },
            ...overrides,
        };
    }

    describe("BotPriorityCalculator", () => {
        describe("calculatePriority", () => {
            it("should return base priority for bot with no configuration", () => {
                const bot = createTestBot();
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBe(20);
            });

            it("should calculate priority with explicit config priority", () => {
                const bot = createTestBot({
                    priority: 5,
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBe(5 * PRIORITY_WEIGHTS.CONFIG_PRIORITY);
            });

            it("should add role weight for arbitrator", () => {
                const bot = createTestBot({
                    role: "arbitrator",
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBe(ROLE_WEIGHTS.ARBITRATOR * PRIORITY_WEIGHTS.ROLE_WEIGHT);
            });

            it("should add role weight for leader", () => {
                const bot = createTestBot({
                    role: "leader",
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBe(ROLE_WEIGHTS.LEADER * PRIORITY_WEIGHTS.ROLE_WEIGHT);
            });

            it("should add role weight for coordinator", () => {
                const bot = createTestBot({
                    role: "coordinator",
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBe(ROLE_WEIGHTS.COORDINATOR * PRIORITY_WEIGHTS.ROLE_WEIGHT);
            });

            it("should add role weight for specialist", () => {
                const bot = createTestBot({
                    role: "specialist",
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBe(ROLE_WEIGHTS.SPECIALIST * PRIORITY_WEIGHTS.ROLE_WEIGHT);
            });

            it("should add role weight for member", () => {
                const bot = createTestBot({
                    role: "member",
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBe(ROLE_WEIGHTS.MEMBER * PRIORITY_WEIGHTS.ROLE_WEIGHT);
            });

            it("should use default role weight for unknown role", () => {
                const bot = createTestBot({
                    role: "unknown",
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBe(ROLE_WEIGHTS.DEFAULT * PRIORITY_WEIGHTS.ROLE_WEIGHT);
            });

            it("should calculate pattern specificity for exact match patterns", () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/transaction/completed" });

                const priority = calculator.calculatePriority(bot, event);

                // Should include pattern specificity score
                const expectedPatternScore = (
                    3 * PATTERN_SCORING.EXACT_MATCH_SCORE + // 3 exact segments
                    3 * PATTERN_SCORING.SEGMENT_SCORE // 3 total segments
                ) * PRIORITY_WEIGHTS.PATTERN_SPECIFICITY;

                expect(priority).toBeGreaterThanOrEqual(expectedPatternScore);
            });

            it("should penalize wildcard patterns", () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/*/completed",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                // Should include pattern specificity with wildcard penalty
                const expectedPatternScore = (
                    2 * PATTERN_SCORING.EXACT_MATCH_SCORE + // 2 exact segments
                    3 * PATTERN_SCORING.SEGMENT_SCORE - // 3 total segments
                    1 * PATTERN_SCORING.WILDCARD_PENALTY // 1 wildcard
                ) * PRIORITY_WEIGHTS.PATTERN_SPECIFICITY;

                expect(priority).toBeGreaterThanOrEqual(expectedPatternScore);
            });

            it("should calculate domain match score", () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/payment/processed" });

                const priority = calculator.calculatePriority(bot, event);

                // Should include domain match for "finance"
                expect(priority).toBeGreaterThan(0);
            });

            it("should combine all priority factors", () => {
                const bot = createTestBot({
                    priority: 3,
                    role: "leader",
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/transaction/completed" });

                const priority = calculator.calculatePriority(bot, event);

                const expectedBase =
                    3 * PRIORITY_WEIGHTS.CONFIG_PRIORITY +
                    ROLE_WEIGHTS.LEADER * PRIORITY_WEIGHTS.ROLE_WEIGHT;

                expect(priority).toBeGreaterThan(expectedBase);
            });

            it("should ensure non-negative priority", () => {
                const bot = createTestBot({
                    priority: -100, // Negative priority
                });
                const event = createTestEvent();

                const priority = calculator.calculatePriority(bot, event);

                expect(priority).toBeGreaterThanOrEqual(0);
            });
        });

        describe("sortByPriority", () => {
            it("should sort bots by priority (highest first)", () => {
                const bot1 = createTestBot({ id: "bot1", priority: 1 });
                const bot2 = createTestBot({ id: "bot2", priority: 3 });
                const bot3 = createTestBot({ id: "bot3", priority: 2 });
                const event = createTestEvent();

                const sorted = calculator.sortByPriority([bot1, bot2, bot3], event);

                expect(sorted[0].id).toBe("bot2"); // Highest priority
                expect(sorted[1].id).toBe("bot3"); // Medium priority
                expect(sorted[2].id).toBe("bot1"); // Lowest priority
            });

            it("should handle empty bot array", () => {
                const event = createTestEvent();

                const sorted = calculator.sortByPriority([], event);

                expect(sorted).toEqual([]);
            });

            it("should handle single bot", () => {
                const bot = createTestBot();
                const event = createTestEvent();

                const sorted = calculator.sortByPriority([bot], event);

                expect(sorted).toHaveLength(1);
                expect(sorted[0]).toBe(bot);
            });
        });
    });

    describe("DefaultDecisionMaker", () => {
        describe("decide", () => {
            it("should handle bot with matching behavior", async () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                                action: {
                                    type: "invoke",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/transaction/completed" });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(true);
                expect(decision.priority).toBeGreaterThan(0);
                expect(decision.response?.progression).toBe("continue");
            });

            it("should handle arbitrator bot even without matching behavior", async () => {
                const bot = createTestBot({
                    role: "arbitrator",
                    config: {
                        __version: "1.0",
                        resources: [],
                        agentSpec: {
                            behaviors: [],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent();
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(true);
                expect(decision.response?.progression).toBe("continue");
            });

            it("should not handle bot without matching behavior", async () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "chat/message/sent" });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(false);
            });

            it("should handle wildcard pattern matches", async () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/*",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/transaction" });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(true);
            });

            it("should handle catch-all pattern", async () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "#",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent();
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(true);
            });

            it("should block event progression for exclusive behaviors", async () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                    progression: {
                                        exclusive: true,
                                    },
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/transaction/completed" });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(true);
                expect(decision.response?.progression).toBe("block");
            });

            it("should block event progression for routine actions", async () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                                action: {
                                    type: "routine",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/transaction/completed" });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(true);
                expect(decision.response?.progression).toBe("block");
            });

            it("should continue event progression for invoke actions", async () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                                action: {
                                    type: "invoke",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/transaction/completed" });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(true);
                expect(decision.response?.progression).toBe("continue");
            });

            it("should not handle when event progression is blocked", async () => {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({
                    type: "finance/transaction/completed",
                    progression: {
                        state: "block",
                        processedBy: [],
                    },
                });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(false);
            });

            it("should generate appropriate decision reasons", async () => {
                const bot = createTestBot({
                    id: "test-bot",
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: "finance/transaction/completed",
                                },
                                action: {
                                    type: "invoke",
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: "finance/transaction/completed" });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.response?.reason).toContain("test-bot");
                expect(decision.response?.reason).toContain("finance/transaction/completed");
                expect(decision.response?.reason).toContain("invoke action");
            });

            it("should handle errors gracefully", async () => {
                const bot = createTestBot({
                    config: null as any, // Invalid config to trigger error
                });
                const event = createTestEvent();
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(false);
                expect(decision.response?.progression).toBe("continue");
                expect(decision.response?.reason).toContain("Decision error");
            });
        });

        describe("pattern matching", () => {
            async function testPatternMatching(pattern: string, eventType: string, shouldMatch: boolean) {
                const bot = createTestBot({
                    config: {
                        agentSpec: {
                            behaviors: [{
                                trigger: {
                                    topic: pattern,
                                },
                            }],
                        },
                    } as BotConfigObject,
                });
                const event = createTestEvent({ type: eventType });
                const context: BotDecisionContext = { bot, event };

                const decision = await decisionMaker.decide(context);

                expect(decision.shouldHandle).toBe(shouldMatch);
            }

            it("should match exact patterns", async () => {
                await testPatternMatching("finance/transaction/completed", "finance/transaction/completed", true);
            });

            it("should not match different exact patterns", async () => {
                await testPatternMatching("finance/transaction/completed", "finance/payment/completed", false);
            });

            it("should match wildcard patterns", async () => {
                await testPatternMatching("finance/*", "finance/transaction", true);
                await testPatternMatching("finance/*", "finance/payment", true);
            });

            it("should not match wildcard patterns outside scope", async () => {
                await testPatternMatching("finance/*", "chat/message", false);
            });

            it("should match catch-all pattern", async () => {
                await testPatternMatching("#", "any/event/type", true);
                await testPatternMatching("#", "completely/different", true);
            });

            it("should handle malformed patterns", async () => {
                await testPatternMatching("", "finance/transaction", false);
                await testPatternMatching("finance/", "finance/transaction", false);
            });
        });
    });
});
