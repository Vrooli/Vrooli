import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { PromptService } from "./promptService.js";
import { DbProvider } from "../../db/provider.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../../validators/permissions.js";
import { CustomError } from "../../events/error.js";
import { ResourceSubType } from "@vrooli/shared";
import * as fs from "fs/promises";

// Mock dependencies
vi.mock("../../db/provider.js");
vi.mock("../../utils/getAuthenticatedData.js");
vi.mock("../../validators/permissions.js");
vi.mock("../../events/logger.js");
vi.mock("fs/promises");

describe("PromptService", () => {
    let service: PromptService;
    
    beforeEach(() => {
        vi.clearAllMocks();
        service = new PromptService(undefined, {
            templateDirectory: "/templates",
            defaultTemplate: "default.txt",
            enableTemplateCache: true,
            userPromptCacheTTL: 5000,
        });
    });
    
    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("loadTemplate", () => {
        it("should load file template when identifier doesn't start with 'prompt:'", async () => {
            const mockTemplate = "Hello {{ROLE}}!";
            vi.mocked(fs.readFile).mockResolvedValue(mockTemplate);
            
            const result = await service.loadTemplate("test.txt");
            
            expect(result).toBe(mockTemplate);
            expect(fs.readFile).toHaveBeenCalledWith("/templates/test.txt", "utf-8");
        });

        it("should load user prompt when identifier starts with 'prompt:'", async () => {
            const mockPromptData = {
                id: "123",
                config: {
                    props: {
                        template: "User prompt: {{GOAL}}",
                    },
                },
                translations: [{
                    language: "en",
                    name: "Test Prompt",
                    description: "A test prompt",
                }],
                versionLabel: "1.0.0",
            };
            
            // Mock database response
            const mockDb = {
                resource_version: {
                    findUnique: vi.fn().mockResolvedValue(mockPromptData),
                },
            };
            vi.mocked(DbProvider.get).mockReturnValue(mockDb as any);
            
            // Mock permission checks
            vi.mocked(getAuthenticatedData).mockResolvedValue({ "123": { __typename: "ResourceVersion", id: "123" } });
            vi.mocked(permissionsCheck).mockResolvedValue(true);
            
            const result = await service.loadTemplate("prompt:123", { id: "user1", languages: ["en"] } as any);
            
            expect(result).toBe("User prompt: {{GOAL}}");
            expect(mockDb.resource_version.findUnique).toHaveBeenCalledWith(
                expect.objectContaining({
                    where: {
                        id: "123",
                        resourceSubType: "StandardPrompt",
                        isDeleted: false,
                    },
                }),
            );
        });

        it("should throw error when user prompt not found", async () => {
            const mockDb = {
                resource_version: {
                    findUnique: vi.fn().mockResolvedValue(null),
                },
            };
            vi.mocked(DbProvider.get).mockReturnValue(mockDb as any);
            
            await expect(service.loadTemplate("prompt:999")).rejects.toThrow(CustomError);
        });

        it("should check permissions for private prompts", async () => {
            const mockPromptData = {
                id: "123",
                isPrivate: true,
                config: {
                    props: {
                        template: "Private prompt",
                    },
                },
                translations: [{
                    language: "en",
                    name: "Private Prompt",
                }],
                versionLabel: "1.0.0",
            };
            
            const mockDb = {
                resource_version: {
                    findUnique: vi.fn().mockResolvedValue(mockPromptData),
                },
            };
            vi.mocked(DbProvider.get).mockReturnValue(mockDb as any);
            
            vi.mocked(getAuthenticatedData).mockResolvedValue({ "123": { __typename: "ResourceVersion", id: "123" } });
            vi.mocked(permissionsCheck).mockRejectedValue(new CustomError("0297", "Unauthorized"));
            
            await expect(service.loadTemplate("prompt:123", { id: "user1", languages: ["en"] } as any))
                .rejects.toThrow(CustomError);
            
            expect(permissionsCheck).toHaveBeenCalled();
        });

        it("should use cached user prompt if valid", async () => {
            const mockPromptData = {
                id: "123",
                config: {
                    props: {
                        template: "Cached prompt",
                    },
                },
                translations: [{
                    language: "en",
                    name: "Test Prompt",
                }],
                versionLabel: "1.0.0",
            };
            
            const mockDb = {
                resource_version: {
                    findUnique: vi.fn().mockResolvedValue(mockPromptData),
                },
            };
            vi.mocked(DbProvider.get).mockReturnValue(mockDb as any);
            vi.mocked(getAuthenticatedData).mockResolvedValue({ "123": { __typename: "ResourceVersion", id: "123" } });
            vi.mocked(permissionsCheck).mockResolvedValue(true);
            
            // First call - should hit database
            const result1 = await service.loadTemplate("prompt:123", { id: "user1", languages: ["en"] } as any);
            expect(result1).toBe("Cached prompt");
            expect(mockDb.resource_version.findUnique).toHaveBeenCalledTimes(1);
            
            // Second call - should use cache
            const result2 = await service.loadTemplate("prompt:123", { id: "user1", languages: ["en"] } as any);
            expect(result2).toBe("Cached prompt");
            expect(mockDb.resource_version.findUnique).toHaveBeenCalledTimes(1); // Still only called once
        });
    });

    describe("buildSystemMessage", () => {
        it("should build message with default template", async () => {
            const mockTemplate = "Goal: {{GOAL}}, Role: {{ROLE}}";
            vi.mocked(fs.readFile).mockResolvedValue(mockTemplate);
            
            const context = {
                goal: "Test goal",
                bot: {
                    id: "bot1",
                    meta: { role: "assistant" },
                } as any,
                convoConfig: {} as any,
            };
            
            const result = await service.buildSystemMessage(context);
            
            expect(result).toContain("Welcome to Vrooli");
            expect(result).toContain("Goal: Test goal");
            expect(result).toContain("Role: assistant");
        });

        it("should apply user inputs for user prompts", async () => {
            const mockPromptData = {
                id: "123",
                config: {
                    props: {
                        template: "Hello {{userName}}, your task is: {{GOAL}}",
                    },
                },
                translations: [{
                    language: "en",
                    name: "User Input Prompt",
                }],
                versionLabel: "1.0.0",
            };
            
            const mockDb = {
                resource_version: {
                    findUnique: vi.fn().mockResolvedValue(mockPromptData),
                },
            };
            vi.mocked(DbProvider.get).mockReturnValue(mockDb as any);
            vi.mocked(getAuthenticatedData).mockResolvedValue({ "123": { __typename: "ResourceVersion", id: "123" } });
            vi.mocked(permissionsCheck).mockResolvedValue(true);
            
            const context = {
                goal: "Complete task",
                bot: {
                    id: "bot1",
                    meta: { role: "assistant" },
                } as any,
                convoConfig: {} as any,
                userInputs: {
                    userName: "Alice",
                },
            };
            
            const result = await service.buildSystemMessage(context, {
                templateIdentifier: "prompt:123",
                userData: { id: "user1", languages: ["en"] } as any,
            });
            
            expect(result).toContain("Hello Alice");
            expect(result).toContain("Complete task");
        });
    });

    describe("extractTemplateFromConfig", () => {
        it("should extract template from various config locations", () => {
            // Test props.template
            expect((service as any).extractTemplateFromConfig({
                props: { template: "From props.template" },
            })).toBe("From props.template");
            
            // Test props.content
            expect((service as any).extractTemplateFromConfig({
                props: { content: "From props.content" },
            })).toBe("From props.content");
            
            // Test schema as JSON with template
            expect((service as any).extractTemplateFromConfig({
                schema: JSON.stringify({ template: "From schema.template" }),
            })).toBe("From schema.template");
            
            // Test schema as plain string
            expect((service as any).extractTemplateFromConfig({
                schema: "Plain schema string",
            })).toBe("Plain schema string");
            
            // Test fallback
            expect((service as any).extractTemplateFromConfig({}))
                .toBe("Your primary goal is: {{GOAL}}. Please act according to your role: {{ROLE}}. Critical: Prompt template file not found.");
        });
    });

    describe("applyUserInputs", () => {
        it("should replace various placeholder formats", () => {
            const template = "Hello {{name}}, your {role} is ${task}";
            const inputs = {
                name: "Alice",
                role: "developer",
                task: "coding",
            };
            
            const result = (service as any).applyUserInputs(template, inputs);
            
            expect(result).toBe("Hello Alice, your developer is coding");
        });

        it("should handle missing inputs gracefully", () => {
            const template = "Hello {{name}}, welcome {{missing}}!";
            const inputs = {
                name: "Bob",
            };
            
            const result = (service as any).applyUserInputs(template, inputs);
            
            expect(result).toBe("Hello Bob, welcome {{missing}}!");
        });
    });
});
