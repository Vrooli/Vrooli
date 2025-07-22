import {
    BotConfig,
    ChatConfig,
    MessageConfig,
    type BotParticipant,
    type ChatConfigObject,
    type ConversationTrigger,
    type RunState,
    type RuntimeResources,
    type SwarmState,
    type SystemMetadata,
} from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
    ActiveBotGraph,
    CompositeGraph,
    DirectResponderGraph,
} from "./agentGraph.js";

describe("DirectResponderGraph", () => {
    let graph: DirectResponderGraph;
    let mockSwarmState: SwarmState;
    let mockAgents: BotParticipant[];

    beforeEach(() => {
        vi.clearAllMocks();
        graph = new DirectResponderGraph();

        // Create mock agents
        mockAgents = [
            {
                id: "bot1",
                name: "Assistant Bot",
                config: BotConfig.parse({ model: "gpt-4o", agentSpec: { role: "assistant" } }),
                state: "ready",
            },
            {
                id: "bot2",
                name: "Research Bot",
                config: BotConfig.parse({ model: "gpt-4o", agentSpec: { role: "researcher" } }),
                state: "ready",
            },
            {
                id: "bot3",
                name: "Arbitrator Bot",
                config: BotConfig.parse({ model: "gpt-4o", agentSpec: { role: "arbitrator" } }),
                state: "ready",
            },
        ];

        // Create mock swarm state
        mockSwarmState = {
            swarmId: "swarm123",
            version: 1,
            chatConfig: ChatConfig.default().export(),
            execution: {
                status: "running" as RunState,
                agents: mockAgents,
                activeRuns: [],
                startedAt: new Date(),
                lastActivityAt: new Date(),
            },
            resources: {
                allocated: [],
                consumed: { credits: 0, tokens: 0, time: 0 },
                remaining: { credits: 1000, tokens: 10000, time: 300 },
            } as RuntimeResources,
            metadata: {
                createdAt: new Date(),
                lastUpdated: new Date(),
                updatedBy: "system",
                subscribers: new Set(),
            } as SystemMetadata,
        };
    });

    describe("selectResponders", () => {
        it("should return empty responders for non-user_message triggers", async () => {
            const trigger: ConversationTrigger = { type: "start" };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result).toEqual({
                responders: [],
                strategy: "direct_mention",
            });
        });

        it("should return empty responders when no agents exist", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: ["bot1"],
                    }),
                },
            };

            // Remove all agents
            mockSwarmState.execution.agents = [];

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result).toEqual({
                responders: [],
                strategy: "direct_mention",
            });
        });

        it("should return empty responders when no respondingBots specified", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: undefined,
                },
            };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result).toEqual({
                responders: [],
                strategy: "direct_mention",
            });
        });

        it("should return all agents when @all is specified", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: ["@all"],
                    }),
                },
            };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result).toEqual({
                responders: mockAgents,
                strategy: "direct_mention",
            });
        });

        it("should return specific bots when IDs are provided", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: ["bot1", "bot3"],
                    }),
                },
            };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(2);
            expect(result.responders.map(bot => bot.id)).toEqual(["bot1", "bot3"]);
            expect(result.strategy).toBe("direct_mention");
        });

        it("should filter out non-existent bot IDs", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: ["bot1", "nonexistent", "bot2"],
                    }),
                },
            };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(2);
            expect(result.responders.map(bot => bot.id)).toEqual(["bot1", "bot2"]);
            expect(result.strategy).toBe("direct_mention");
        });

        it("should remove duplicate bot IDs", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: ["bot1", "bot1", "bot2", "bot1"],
                    }),
                },
            };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(2);
            expect(result.responders.map(bot => bot.id)).toEqual(["bot1", "bot2"]);
            expect(result.strategy).toBe("direct_mention");
        });

        it("should handle empty respondingBots array", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: [],
                    }),
                },
            };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result).toEqual({
                responders: [],
                strategy: "direct_mention",
            });
        });
    });
});

describe("ActiveBotGraph", () => {
    let graph: ActiveBotGraph;
    let mockSwarmState: SwarmState;
    let mockAgents: BotParticipant[];
    let consoleSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
        vi.clearAllMocks();
        consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => { });

        mockAgents = [
            {
                id: "bot1",
                name: "Assistant Bot",
                config: BotConfig.parse({ model: "gpt-4o", agentSpec: { role: "assistant" } }),
                state: "ready",
            },
            {
                id: "bot2",
                name: "Arbitrator Bot",
                config: BotConfig.parse({ model: "gpt-4o", agentSpec: { role: "arbitrator" } }),
                state: "ready",
            },
            {
                id: "bot3",
                name: "Leader Bot",
                config: BotConfig.parse({ model: "gpt-4o", agentSpec: { role: "leader" } }),
                state: "ready",
            },
        ];

        mockSwarmState = {
            swarmId: "swarm123",
            version: 1,
            chatConfig: ChatConfig.default().export(),
            execution: {
                status: "running" as RunState,
                agents: mockAgents,
                activeRuns: [],
                startedAt: new Date(),
                lastActivityAt: new Date(),
            },
            resources: {
                allocated: [],
                consumed: { credits: 0, tokens: 0, time: 0 },
                remaining: { credits: 1000, tokens: 10000, time: 300 },
            } as RuntimeResources,
            metadata: {
                createdAt: new Date(),
                lastUpdated: new Date(),
                updatedBy: "system",
                subscribers: new Set(),
            } as SystemMetadata,
        };
    });

    afterEach(() => {
        consoleSpy.mockRestore();
    });

    describe("selectResponders", () => {
        it("should return arbitrator when no activeBotId is set", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot2");
            expect(result.strategy).toBe("swarm_baton");
            expect(consoleSpy).not.toHaveBeenCalled();
        });

        it("should return active bot when activeBotId is set", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };
            mockSwarmState.chatConfig = {
                ...mockSwarmState.chatConfig,
                activeBotId: "bot1",
            } as ChatConfigObject;

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot1");
            expect(result.strategy).toBe("swarm_baton");
        });

        it("should return empty when activeBotId is set but bot not found", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };
            mockSwarmState.chatConfig = {
                ...mockSwarmState.chatConfig,
                activeBotId: "nonexistent",
            } as ChatConfigObject;

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(0);
            expect(result.strategy).toBe("swarm_baton");
        });

        it("should find arbitrator with exact role match", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot2");
            expect(result.responders[0].config?.agentSpec?.role).toBe("arbitrator");
        });

        it("should find secondary arbitrator roles when no exact arbitrator", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };

            // Remove the arbitrator bot
            mockSwarmState.execution.agents = mockAgents.filter(bot => bot.id !== "bot2");

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot3");
            expect(result.responders[0].config?.agentSpec?.role).toBe("leader");
        });

        it("should handle case-insensitive role matching", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };

            // Modify bot role to have different casing
            mockSwarmState.execution.agents[1].config = BotConfig.parse({
                ...mockSwarmState.execution.agents[1].config,
                agentSpec: { role: "ARBITRATOR" },
            });

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot2");
        });

        it("should handle roles with whitespace", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };

            mockSwarmState.execution.agents[1].config = BotConfig.parse({
                ...mockSwarmState.execution.agents[1].config,
                agentSpec: { role: "  arbitrator  " },
            });

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot2");
        });

        it("should warn when no arbitrator found (with warnings enabled)", async () => {
            graph = new ActiveBotGraph(false); // warnings enabled
            const trigger: ConversationTrigger = { type: "start" };

            // Remove all bots with arbitrator roles
            mockSwarmState.execution.agents = [mockAgents[0]]; // Only assistant bot

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(0);
            expect(consoleSpy).toHaveBeenCalledWith(
                expect.stringContaining("Expected an arbitrator participant, but none was found"),
            );
        });

        it("should not warn when no arbitrator found (with warnings suppressed)", async () => {
            graph = new ActiveBotGraph(true); // warnings suppressed
            const trigger: ConversationTrigger = { type: "start" };

            mockSwarmState.execution.agents = [mockAgents[0]]; // Only assistant bot

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(0);
            expect(consoleSpy).not.toHaveBeenCalled();
        });

        it("should return empty when no agents exist", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };

            mockSwarmState.execution.agents = [];

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(0);
            expect(result.strategy).toBe("swarm_baton");
        });

        it("should find delegator as secondary arbitrator role", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };

            // Add a delegator bot and remove arbitrator
            mockSwarmState.execution.agents = [
                mockAgents[0],
                {
                    id: "bot4",
                    name: "Delegator Bot",
                    config: BotConfig.parse({
                        model: "gpt-4o",
                        agentSpec: { role: "delegator" },
                    }),
                    state: "ready",
                },
                mockAgents[2],
            ];

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot4");
            expect(result.responders[0].config?.agentSpec?.role).toBe("delegator");
        });

        it("should find coordinator as secondary arbitrator role", async () => {
            graph = new ActiveBotGraph();
            const trigger: ConversationTrigger = { type: "start" };

            // Add a coordinator bot and remove arbitrator
            mockSwarmState.execution.agents = [
                mockAgents[0],
                {
                    id: "bot5",
                    name: "Coordinator Bot",
                    config: BotConfig.parse({
                        model: "gpt-4o",
                        agentSpec: { role: "coordinator" },
                    }),
                    state: "ready",
                },
            ];

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot5");
            expect(result.responders[0].config?.agentSpec?.role).toBe("coordinator");
        });
    });
});

describe("CompositeGraph", () => {
    let graph: CompositeGraph;
    let mockSwarmState: SwarmState;
    let mockAgents: BotParticipant[];
    let mockDirect: DirectResponderGraph;
    let mockActiveBotGraph: ActiveBotGraph;

    beforeEach(() => {
        vi.clearAllMocks();

        mockAgents = [
            {
                id: "bot1",
                name: "Assistant Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            },
            {
                id: "bot2",
                name: "Arbitrator Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "arbitrator" },
                }),
                state: "ready",
            },
        ];

        mockSwarmState = {
            swarmId: "swarm123",
            version: 1,
            chatConfig: ChatConfig.default().export(),
            execution: {
                status: "running" as RunState,
                agents: mockAgents,
                activeRuns: [],
                startedAt: new Date(),
                lastActivityAt: new Date(),
            },
            resources: {
                allocated: [],
                consumed: { credits: 0, tokens: 0, time: 0 },
                remaining: { credits: 1000, tokens: 10000, time: 300 },
            } as RuntimeResources,
            metadata: {
                createdAt: new Date(),
                lastUpdated: new Date(),
                updatedBy: "system",
                subscribers: new Set(),
            } as SystemMetadata,
        };

        mockDirect = new DirectResponderGraph();
        mockActiveBotGraph = new ActiveBotGraph(true);
        graph = new CompositeGraph(mockDirect, mockActiveBotGraph);
    });

    describe("selectResponders", () => {
        it("should return direct responders when available (priority 1)", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: ["bot1"],
                    }),
                },
            };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot1");
            expect(result.strategy).toBe("direct_mention");
        });

        it("should use active bot when no direct responders (priority 2)", async () => {
            const trigger: ConversationTrigger = { type: "start" };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot2"); // arbitrator
            expect(result.strategy).toBe("swarm_baton");
        });

        it("should respect activeBotId even when bot is missing", async () => {
            const trigger: ConversationTrigger = { type: "start" };
            mockSwarmState.chatConfig = {
                ...mockSwarmState.chatConfig,
                activeBotId: "nonexistent",
            } as ChatConfigObject;

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(0);
            expect(result.strategy).toBe("swarm_baton");
        });

        it("should fall back to first bot when no other strategies work (priority 3)", async () => {
            const trigger: ConversationTrigger = { type: "start" };

            // Remove arbitrator role to make active bot graph return empty
            mockSwarmState.execution.agents = [
                {
                    id: "bot1",
                    name: "Regular Bot",
                    config: BotConfig.parse({
                        model: "gpt-4o",
                        agentSpec: { role: "worker" },
                    }),
                    state: "ready",
                },
                {
                    id: "bot2",
                    name: "Another Bot",
                    config: BotConfig.parse({
                        model: "gpt-4o",
                        agentSpec: { role: "helper" },
                    }),
                    state: "ready",
                },
            ];

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot1");
            expect(result.strategy).toBe("fallback");
        });

        it("should return empty when no bots exist", async () => {
            const trigger: ConversationTrigger = { type: "start" };
            mockSwarmState.execution.agents = [];

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(0);
            expect(result.strategy).toBe("fallback");
        });

        it("should prefer direct over active bot", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: ["bot1"],
                    }),
                },
            };

            // Set activeBotId, but direct should still win
            mockSwarmState.chatConfig = {
                ...mockSwarmState.chatConfig,
                activeBotId: "bot2",
            } as ChatConfigObject;

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot1");
            expect(result.strategy).toBe("direct_mention");
        });

        it("should prefer active bot over fallback", async () => {
            const trigger: ConversationTrigger = { type: "start" };
            mockSwarmState.chatConfig = {
                ...mockSwarmState.chatConfig,
                activeBotId: "bot1",
            } as ChatConfigObject;

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0].id).toBe("bot1");
            expect(result.strategy).toBe("swarm_baton");
        });

        it("should handle @all in direct responders", async () => {
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: MessageConfig.parse({
                        respondingBots: ["@all"],
                    }),
                },
            };

            const result = await graph.selectResponders(mockSwarmState, trigger);

            expect(result.responders).toHaveLength(2);
            expect(result.responders.map(bot => bot.id)).toEqual(["bot1", "bot2"]);
            expect(result.strategy).toBe("direct_mention");
        });
    });
});

describe("AgentGraph integration scenarios", () => {
    let graph: CompositeGraph;
    let mockSwarmState: SwarmState;

    beforeEach(() => {
        vi.clearAllMocks();

        const mockAgents: BotParticipant[] = [
            {
                id: "researcher",
                name: "Research Assistant",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: {
                        role: "researcher",
                    },
                }),
                state: "ready",
            },
            {
                id: "writer",
                name: "Content Writer",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: {
                        role: "writer",
                    },
                }),
                state: "ready",
            },
            {
                id: "coordinator",
                name: "Team Coordinator",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: {
                        role: "arbitrator",
                    },
                }),
                state: "ready",
            },
        ];

        mockSwarmState = {
            swarmId: "team-swarm",
            version: 1,
            chatConfig: ChatConfig.default().export(),
            execution: {
                status: "running" as RunState,
                agents: mockAgents,
                activeRuns: [],
                startedAt: new Date(),
                lastActivityAt: new Date(),
            },
            resources: {
                allocated: [],
                consumed: { credits: 0, tokens: 0, time: 0 },
                remaining: { credits: 2000, tokens: 20000, time: 600 },
            } as RuntimeResources,
            metadata: {
                createdAt: new Date(),
                lastUpdated: new Date(),
                updatedBy: "system",
                subscribers: new Set(),
            } as SystemMetadata,
        };

        graph = new CompositeGraph();
    });

    it("should handle team handoff workflow", async () => {
        // 1. Initial message - coordinator starts
        const startTrigger: ConversationTrigger = { type: "start" };
        const startResult = await graph.selectResponders(mockSwarmState, startTrigger);

        expect(startResult.responders[0].id).toBe("coordinator");
        expect(startResult.strategy).toBe("swarm_baton");

        // 2. Coordinator delegates to researcher
        mockSwarmState.chatConfig = {
            ...mockSwarmState.chatConfig,
            activeBotId: "researcher",
        } as ChatConfigObject;

        const researchTrigger: ConversationTrigger = { type: "continue", lastEvent: {} as any };
        const researchResult = await graph.selectResponders(mockSwarmState, researchTrigger);

        expect(researchResult.responders[0].id).toBe("researcher");
        expect(researchResult.strategy).toBe("swarm_baton");

        // 3. After research, hand off to writer
        mockSwarmState.chatConfig = {
            ...mockSwarmState.chatConfig,
            activeBotId: "writer",
        } as ChatConfigObject;

        const writeTrigger: ConversationTrigger = { type: "continue", lastEvent: {} as any };
        const writeResult = await graph.selectResponders(mockSwarmState, writeTrigger);

        expect(writeResult.responders[0].id).toBe("writer");
        expect(writeResult.strategy).toBe("swarm_baton");

        // 4. User directly asks coordinator for status
        const directTrigger: ConversationTrigger = {
            type: "user_message",
            message: {
                config: {
                    respondingBots: ["coordinator"],
                },
            },
        };

        const directResult = await graph.selectResponders(mockSwarmState, directTrigger);

        expect(directResult.responders[0].id).toBe("coordinator");
        expect(directResult.strategy).toBe("direct_mention");
    });

    it("should handle missing bot gracefully in active workflow", async () => {
        // Set active bot to non-existent ID
        mockSwarmState.chatConfig = {
            ...mockSwarmState.chatConfig,
            activeBotId: "nonexistent-bot",
        } as ChatConfigObject;

        const trigger: ConversationTrigger = { type: "continue", lastEvent: {} as any };
        const result = await graph.selectResponders(mockSwarmState, trigger);

        // Should return empty since activeBotId was set but bot doesn't exist
        expect(result.responders).toHaveLength(0);
        expect(result.strategy).toBe("swarm_baton");
    });

    it("should handle multi-bot direct addressing", async () => {
        const trigger: ConversationTrigger = {
            type: "user_message",
            message: {
                config: {
                    respondingBots: ["researcher", "writer"],
                },
            },
        };

        const result = await graph.selectResponders(mockSwarmState, trigger);

        expect(result.responders).toHaveLength(2);
        expect(result.responders.map(bot => bot.id).sort()).toEqual(["researcher", "writer"]);
        expect(result.strategy).toBe("direct_mention");
    });

    it("should handle complex agent hierarchies", async () => {
        // Add agents with more complex role hierarchies
        const complexAgents: BotParticipant[] = [
            {
                id: "senior-lead",
                name: "Senior Lead",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "leader" },
                }),
                state: "ready",
            },
            {
                id: "project-manager",
                name: "Project Manager",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "coordinator" },
                }),
                state: "ready",
            },
            {
                id: "specialist",
                name: "Specialist",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "specialist" },
                }),
                state: "ready",
            },
        ];

        mockSwarmState.execution.agents = complexAgents;

        const trigger: ConversationTrigger = { type: "start" };
        const result = await graph.selectResponders(mockSwarmState, trigger);

        // Should pick leader as secondary arbitrator role
        expect(result.responders[0].id).toBe("senior-lead");
        expect(result.strategy).toBe("swarm_baton");
    });
});
