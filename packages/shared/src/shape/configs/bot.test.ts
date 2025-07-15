/* eslint-disable @typescript-eslint/ban-ts-comment */
import jexl from "jexl";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { botConfigFixtures } from "../../__test/fixtures/config/botConfigFixtures.js";
import { OpenAIModel } from "../../ai/services.js";
import { type User } from "../../api/types.js";
import { type AgentSpec, type BehaviourSpec, BotConfig, type BotConfigObject, getModelDescription, getModelName, type LlmModel, type ResourceSpec } from "./bot.js";

const LATEST_VERSION_STRING = "1.0"; // Consistent version string

// Valid bot data for new BotConfig structure
const validBotSettingsObject: BotConfigObject = botConfigFixtures.complete;

// Test data will use BotConfigObject | null | undefined for botSettings
const validUserWithBotSettings: Pick<User, "botSettings"> = {
    botSettings: validBotSettingsObject, // Direct object, no JSON.stringify
};

describe("BotConfig", () => {
    let consoleErrorSpy: any;

    beforeAll(() => {
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {
            // Mock implementation for testing
        });
    });

    beforeEach(() => {
        consoleErrorSpy.mockClear();
    });

    afterAll(() => {
        consoleErrorSpy.mockRestore();
    });

    it("parses valid bot settings object correctly", () => {
        const botConfigInstance = BotConfig.parse(validUserWithBotSettings, console);
        expect(botConfigInstance.export()).toEqual(validBotSettingsObject);
        expect(consoleErrorSpy).not.toHaveBeenCalled();
    });

    describe("handles botSettings property being null or undefined", () => {
        it("botSettings is null", () => {
            const userData: Pick<User, "botSettings"> = {
                botSettings: null,
            };
            const botConfigInstance = BotConfig.parse(userData, console);
            // Should return default config because botSettings is null
            // parseBase handles null data by defaulting fields like __version and resources,
            // and BotConfig factory defaults persona.
            expect(botConfigInstance.export()).toEqual(BotConfig.default().export());
            // No error expected as null is a permissible value for botSettings (Maybe<BotConfigObject>)
            // and parseBase handles it gracefully by falling back to defaults.
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("botSettings is undefined", () => {
            const userData: Pick<User, "botSettings"> = {
                botSettings: undefined,
            };
            const botConfigInstance = BotConfig.parse(userData, console);
            // Should return default config because botSettings is undefined
            expect(botConfigInstance.export()).toEqual(BotConfig.default().export());
            // No error expected as undefined is permissible and handled by parseBase.
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });
    });

    describe("handles invalid overall input for the User-like object passed to parse", () => {
        it("User-like object is null", () => {
            const nullUserData = null as unknown as Pick<User, "botSettings">;
            const botConfigInstance = BotConfig.parse(nullUserData, console);
            expect(botConfigInstance.export()).toEqual(BotConfig.default().export());
            // If bot is null, bot?.botSettings is undefined. parseBase receives undefined and defaults.
            // No error is logged in this path.
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("User-like object is undefined", () => {
            const undefinedUserData = undefined as unknown as Pick<User, "botSettings">;
            const botConfigInstance = BotConfig.parse(undefinedUserData, console);
            expect(botConfigInstance.export()).toEqual(BotConfig.default().export());
            // If bot is undefined, bot?.botSettings is undefined. parseBase receives undefined and defaults.
            // No error is logged in this path.
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        // Removed test for "array input for User object" as it's less a direct concern of BotConfig.parse 
        // and more about how Pick<User, "botSettings"> would be constructed or type-checked upstream.
        // BotConfig.parse expects an object (or null/undefined for the object itself based on tests above).
    });

    it("handles botSettings being an empty object literal '{}'", () => {
        const userData: Pick<User, "botSettings"> = {
            botSettings: {} as BotConfigObject, // Valid: empty object, but missing __version
        };
        const botConfigInstance = BotConfig.parse(userData, console);
        // parseBase will get this empty object. `data.__version` will be undefined.
        // So, `__version` will be LATEST_CONFIG_VERSION.
        // `resources` will be [].
        // Effectively, an empty object becomes a default config.
        expect(botConfigInstance.export()).toEqual(BotConfig.default().export());
        expect(consoleErrorSpy).not.toHaveBeenCalled(); // No error, as it's gracefully defaulted.
    });

    it("retains valid fields from botSettings object and fills defaults for missing ones", () => {
        const partialBotSettingsObject: Partial<BotConfigObject> = {
            // __version is missing, should be defaulted by parseBase
            model: "retained-model",
            // maxTokens is missing, should be undefined in BotConfig constructor
            // resources are missing, should be defaulted by parseBase to []
        };
        const userData: Pick<User, "botSettings"> = {
            botSettings: partialBotSettingsObject as BotConfigObject, // Cast because it's partial
        };
        const botConfigInstance = BotConfig.parse(userData, console);

        const expectedExport: BotConfigObject = {
            __version: LATEST_VERSION_STRING, // Defaulted by parseBase
            model: "retained-model",
            maxTokens: undefined, // BotConfig constructor specific
            resources: [], // Defaulted by parseBase
            agentSpec: undefined,
        };
        expect(botConfigInstance.export()).toEqual(expectedExport);
        expect(consoleErrorSpy).not.toHaveBeenCalled();
    });

    it("parses a complete botSettings object correctly", () => {
        // This is similar to the first test but re-confirms with a full object.
        const fullBotSettings = botConfigFixtures.complete;
        const userData: Pick<User, "botSettings"> = {
            botSettings: fullBotSettings,
        };
        const botConfigInstance = BotConfig.parse(userData, console);
        expect(botConfigInstance.export()).toEqual(fullBotSettings);
        expect(consoleErrorSpy).not.toHaveBeenCalled();
    });


    describe("agentSpec handling", () => {
        it("should parse bot settings with complete agentSpec correctly", () => {
            const completeAgentSpec: AgentSpec = {
                goal: "Orchestrate complex workflows",
                role: "coordinator",
                subscriptions: ["swarm/goal/created", "run/completed"],
                behaviors: [
                    {
                        trigger: { topic: "swarm/goal/created" },
                        action: {
                            type: "routine",
                            label: "Task Decomposer",
                            routineId: "1234567890123456789",
                            inputMap: { goalDescription: "event.data.description" },
                        },
                        qos: 1,
                    },
                    {
                        trigger: {
                            topic: "run/completed",
                            when: "event.data.status === 'success'",
                        },
                        action: {
                            type: "invoke",
                            purpose: "Analyze completion and determine next steps",
                        },
                        qos: 0,
                    },
                ],
                norms: [
                    { modality: "obligation", target: "monitor-progress" },
                    { modality: "prohibition", target: "exceed-budget" },
                ],
                resources: [
                    { type: "routine", label: "Task Decomposer", id: "1234567890123456789" },
                    { type: "document", label: "Workflow Best Practices", id: "doc-123" },
                    { type: "tool", label: "Progress Tracker", permissions: ["read", "write"] },
                    { type: "blackboard", label: "Swarm Memory" },
                    { type: "link", label: "Documentation", id: "https://docs.example.com" },
                ],
            };

            const settingsWithAgentSpec: BotConfigObject = {
                ...botConfigFixtures.minimal,
                model: "gpt-4",
                agentSpec: completeAgentSpec,
            };

            const userData: Pick<User, "botSettings"> = {
                botSettings: settingsWithAgentSpec,
            };

            const botConfig = BotConfig.parse(userData, console);
            expect(botConfig.agentSpec).toEqual(completeAgentSpec);
            expect(botConfig.export().agentSpec).toEqual(completeAgentSpec);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("should handle missing agentSpec gracefully", () => {
            const settingsWithoutAgentSpec: BotConfigObject = {
                ...botConfigFixtures.minimal,
                model: "gpt-4",
            };

            const userData: Pick<User, "botSettings"> = {
                botSettings: settingsWithoutAgentSpec,
            };

            const botConfig = BotConfig.parse(userData, console);
            expect(botConfig.agentSpec).toBeUndefined();
            expect(botConfig.export().agentSpec).toBeUndefined();
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("should handle partial agentSpec correctly", () => {
            const partialAgentSpec: AgentSpec = {
                goal: "Monitor system health",
                role: "monitor",
                subscriptions: ["run/failed", "step/failed"],
                // Missing behaviors, norms, and resources
            };

            const settingsWithPartialAgentSpec: BotConfigObject = {
                ...botConfigFixtures.minimal,
                agentSpec: partialAgentSpec,
            };

            const userData: Pick<User, "botSettings"> = {
                botSettings: settingsWithPartialAgentSpec,
            };

            const botConfig = BotConfig.parse(userData, console);
            expect(botConfig.agentSpec).toEqual(partialAgentSpec);
            expect(botConfig.agentSpec?.behaviors).toBeUndefined();
            expect(botConfig.agentSpec?.norms).toBeUndefined();
            expect(botConfig.agentSpec?.resources).toBeUndefined();
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("should preserve agentSpec with empty arrays", () => {
            const agentSpecWithEmptyArrays: AgentSpec = {
                goal: "Test agent",
                role: "test",
                subscriptions: [],
                behaviors: [],
                norms: [],
                resources: [],
            };

            const settingsWithEmptyArrays: BotConfigObject = {
                ...botConfigFixtures.minimal,
                agentSpec: agentSpecWithEmptyArrays,
            };

            const userData: Pick<User, "botSettings"> = {
                botSettings: settingsWithEmptyArrays,
            };

            const botConfig = BotConfig.parse(userData, console);
            expect(botConfig.agentSpec).toEqual(agentSpecWithEmptyArrays);
            expect(botConfig.agentSpec?.subscriptions).toEqual([]);
            expect(botConfig.agentSpec?.behaviors).toEqual([]);
            expect(botConfig.agentSpec?.norms).toEqual([]);
            expect(botConfig.agentSpec?.resources).toEqual([]);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("should handle complex behaviour actions correctly", () => {
            const routineBehaviour: BehaviourSpec = {
                trigger: {
                    topic: "swarm/resource/updated",
                    when: "event.data.remaining < event.data.allocated * 0.1",
                },
                action: {
                    type: "routine",
                    label: "Resource Rebalancer",
                    inputMap: {
                        currentResources: "event.data.remaining",
                        threshold: "event.data.allocated * 0.1",
                    },
                },
                qos: 2,
            };

            const invokeBehaviour: BehaviourSpec = {
                trigger: { topic: "safety/pre_action" },
                action: {
                    type: "invoke",
                    purpose: "Validate action safety and compliance",
                },
            };

            const agentSpec: AgentSpec = {
                behaviors: [routineBehaviour, invokeBehaviour],
            };

            const settings: BotConfigObject = {
                ...botConfigFixtures.minimal,
                agentSpec,
            };

            const userData: Pick<User, "botSettings"> = { botSettings: settings };
            const botConfig = BotConfig.parse(userData, console);

            expect(botConfig.agentSpec?.behaviors).toHaveLength(2);
            expect(botConfig.agentSpec?.behaviors?.[0]).toEqual(routineBehaviour);
            expect(botConfig.agentSpec?.behaviors?.[1]).toEqual(invokeBehaviour);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });

        it("should handle all resource types correctly", () => {
            const resources: ResourceSpec[] = [
                { type: "routine", label: "Data Processor", id: "routine-123", permissions: ["execute"] },
                { type: "document", label: "User Guide", id: "doc-456" },
                { type: "tool", label: "API Client", permissions: ["read", "write", "execute"] },
                { type: "blackboard", label: "Shared Memory", id: "bb-789" },
                { type: "link", label: "External API", id: "https://api.example.com" },
            ];

            const agentSpec: AgentSpec = { resources };
            const settings: BotConfigObject = {
                ...botConfigFixtures.minimal,
                agentSpec,
            };

            const userData: Pick<User, "botSettings"> = { botSettings: settings };
            const botConfig = BotConfig.parse(userData, console);

            expect(botConfig.agentSpec?.resources).toEqual(resources);
            expect(consoleErrorSpy).not.toHaveBeenCalled();
        });
    });

    it("default method should return a BotConfig instance with the latest version and default values", () => {
        const defaultConfig = BotConfig.default();
        expect(defaultConfig).to.be.instanceOf(BotConfig);
        const exportedDefault = defaultConfig.export();
        expect(exportedDefault.__version).toBe(LATEST_VERSION_STRING);
        expect(exportedDefault.model).toBeUndefined();
        expect(exportedDefault.maxTokens).toBeUndefined();
        expect(exportedDefault.resources).toEqual([]);
        expect(exportedDefault.agentSpec).toBeUndefined();
        expect(consoleErrorSpy).not.toHaveBeenCalled();
    });
});

describe("getModelName", () => {
    const mockTranslation = vi.fn();

    beforeEach(() => {
        mockTranslation.mockClear();
    });

    it("should return translated name when option is provided", () => {
        const model: LlmModel = {
            name: "model.gpt4",
            value: "gpt-4",
        };
        mockTranslation.mockReturnValue("GPT-4");

        const result = getModelName(model, mockTranslation);

        expect(result).toBe("GPT-4");
        expect(mockTranslation).toHaveBeenCalledWith("model.gpt4", { ns: "service" });
    });

    it("should return empty string when option is null", () => {
        const result = getModelName(null, mockTranslation);

        expect(result).toBe("");
        expect(mockTranslation).not.toHaveBeenCalled();
    });
});

describe("getModelDescription", () => {
    const mockTranslation = vi.fn();

    beforeEach(() => {
        mockTranslation.mockClear();
    });

    it("should return translated description when option has description", () => {
        const model: LlmModel = {
            name: "model.gpt4",
            description: "model.gpt4.description",
            value: "gpt-4",
        };
        mockTranslation.mockReturnValue("Advanced GPT-4 model");

        const result = getModelDescription(model, mockTranslation);

        expect(result).toBe("Advanced GPT-4 model");
        expect(mockTranslation).toHaveBeenCalledWith("model.gpt4.description", { ns: "service" });
    });

    it("should return empty string when option has no description", () => {
        const model: LlmModel = {
            name: "model.gpt4",
            value: "gpt-4",
        };

        const result = getModelDescription(model, mockTranslation);

        expect(result).toBe("");
        expect(mockTranslation).not.toHaveBeenCalled();
    });
});

describe("OpenAIModel related tests", () => {
    it("should exist (placeholder)", () => {
        expect(OpenAIModel).toBeDefined();
    });
});

describe("jexl trigger evaluation", () => {

    const mockTriggerContext = {
        event: {
            type: "swarm/resource/updated",
            data: {
                remaining: 50,  // Changed to make the test case true
                allocated: 1000,
                newState: "FAILED",
                status: "success",
            },
            timestamp: new Date(),
        },
        swarm: {
            state: "RUNNING",
            resources: {
                allocated: { credits: 1000, tokens: 50000, time: 3600000 },
                consumed: { credits: 850, tokens: 42000, time: 2800000 },
                remaining: { credits: 150, tokens: 8000, time: 800000 },
            },
            agents: 3,
            id: "swarm123",
        },
        bot: {
            id: "bot456",
            performance: {
                tasksCompleted: 25,
                tasksFailled: 3,
                averageCompletionTime: 1500,
                successRate: 0.89,
                resourceEfficiency: 0.75,
            },
        },
    };

    it("should evaluate simple event data expressions correctly", async () => {
        const expression = "event.data.remaining < event.data.allocated * 0.1";
        const result = await jexl.eval(expression, mockTriggerContext);
        expect(result).toBe(true); // 50 < 1000 * 0.1 (100)
    });

    it("should evaluate state comparison expressions correctly", async () => {
        const expression = "event.data.newState == 'FAILED' || event.data.newState == 'TERMINATED'";
        const result = await jexl.eval(expression, mockTriggerContext);
        expect(result).toBe(true); // newState is 'FAILED'
    });

    it("should evaluate nested swarm resource expressions correctly", async () => {
        const expression = "swarm.resources.consumed.credits > swarm.resources.allocated.credits * 0.8";
        const result = await jexl.eval(expression, mockTriggerContext);
        expect(result).toBe(true); // 850 > 1000 * 0.8 (800)
    });

    it("should evaluate bot performance expressions correctly", async () => {
        const expression = "bot.performance.successRate < 0.8";
        const result = await jexl.eval(expression, mockTriggerContext);
        expect(result).toBe(false); // 0.89 is not < 0.8
    });

    it("should evaluate complex combined expressions correctly", async () => {
        const expression = "swarm.agents >= 3 && swarm.state == 'RUNNING'";
        const result = await jexl.eval(expression, mockTriggerContext);
        expect(result).toBe(true); // 3 >= 3 and state is 'RUNNING'
    });

    it("should handle expressions that reference non-existent fields gracefully", async () => {
        const expression = "event.data.nonExistentField == 'test'";
        const result = await jexl.eval(expression, mockTriggerContext);
        expect(result).toBe(false); // undefined == 'test' is false
    });

    it("should work with expressions used in actual agent behaviors", async () => {
        // Test expressions from our actual agent templates
        const testContextForSuccess = {
            ...mockTriggerContext,
            event: {
                ...mockTriggerContext.event,
                data: { status: "success" },
            },
        };

        const expression = "event.data.status == 'success'";  // jexl uses == not ===
        const result = await jexl.eval(expression, testContextForSuccess);
        expect(result).toBe(true);
    });

    it("should handle mathematical operations in expressions", async () => {
        const expression = "event.data.allocated * 0.1";
        const result = await jexl.eval(expression, mockTriggerContext);
        expect(result).toBe(100); // 1000 * 0.1
    });

    it("should validate expressions that would be used in ResourceSpec behaviors", async () => {
        const lowResourceContext = {
            ...mockTriggerContext,
            event: {
                ...mockTriggerContext.event,
                data: {
                    remaining: 50,
                    allocated: 1000,
                },
            },
        };

        const expression = "event.data.remaining < event.data.allocated * 0.1";
        const result = await jexl.eval(expression, lowResourceContext);
        expect(result).toBe(true); // 50 < 100
    });
});
