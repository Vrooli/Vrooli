/**
 * Comprehensive test suite for MCP tools
 * Tests all major functionality of BuiltInTools and SwarmTools classes
 */

import type { SessionUser } from "@vrooli/shared";
import { DEFAULT_LANGUAGE, generatePK, McpToolName, ResourceSubType, ResourceType } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";
import { DbProvider } from "../../db/provider.js";
import {
    defineToolFixtures,
    mockConversationState,
    mockQueueResults,
    mockResourceManageSchema,
    resourceManageFixtures,
    runRoutineFixtures,
    sendMessageFixtures,
    spawnSwarmFixtures,
    swarmToolsFixtures,
} from "./__test/fixtures/mcpToolFixtures.js";
import { BuiltInTools, SwarmTools } from "./tools.js";

// Mock external dependencies before imports
vi.mock("../../events/logger.js");
vi.mock("../../tasks/run/queue.js");
vi.mock("../../tasks/swarm/queue.js");
vi.mock("../../tasks/queues.js");
vi.mock("../../tasks/swarm/process.js");
vi.mock("./schemaLoader.js");
vi.mock("./context.js");
vi.mock("fs");
vi.mock("path");
vi.mock("node:url");

// Mock conversation state store
const mockConversationStore = {
    get: vi.fn(),
    updateTeamConfig: vi.fn(),
    set: vi.fn(),
    delete: vi.fn(),
};

// Mock imports from process run
let mockProcessRun: any;
let mockProcessNewSwarmExecution: any;
let mockActiveSwarmRegistry: any;

describe("BuiltInTools", () => {
    let testUser: SessionUser;
    let adminUser: SessionUser;
    let tools: BuiltInTools;
    let adminTools: BuiltInTools;

    beforeAll(async () => {
        // Create test users
        const users = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
        testUser = {
            ...users.records[0],
            languages: [DEFAULT_LANGUAGE],
            isLoggedIn: true,
            roles: [],
            wallets: [],
            walletIdCurrent: null,
            isAdmin: false,
            theme: "light",
        };
        adminUser = {
            ...users.records[1],
            languages: [DEFAULT_LANGUAGE],
            isLoggedIn: true,
            roles: [],
            wallets: [],
            walletIdCurrent: null,
            isAdmin: true,
            theme: "light",
        };
    });

    afterAll(async () => {
        // Cleanup test data
        await cleanupGroups.userAuth(DbProvider.get());
        await cleanupGroups.chat(DbProvider.get());
        await cleanupGroups.team(DbProvider.get());
        await cleanupGroups.execution(DbProvider.get());
    });

    beforeEach(async () => {
        // Clear all mocks first
        vi.clearAllMocks();

        // Setup mock implementations
        const { logger } = await import("../../events/logger.js");
        vi.mocked(logger).info = vi.fn();
        vi.mocked(logger).error = vi.fn();
        vi.mocked(logger).warn = vi.fn();
        vi.mocked(logger).debug = vi.fn();

        const { processRun } = await import("../../tasks/run/queue.js");
        vi.mocked(processRun).mockResolvedValue(mockQueueResults.runSuccess);

        const { processNewSwarmExecution } = await import("../../tasks/swarm/queue.js");
        vi.mocked(processNewSwarmExecution).mockResolvedValue(mockQueueResults.swarmSuccess);

        const { activeSwarmRegistry } = await import("../../tasks/swarm/process.js");
        vi.mocked(activeSwarmRegistry.get).mockReturnValue(null);
        vi.mocked(activeSwarmRegistry.getOrderedRecords).mockReturnValue([]);

        const { QueueService } = await import("../../tasks/queues.js");
        vi.mocked(QueueService.get).mockReturnValue({
            init: vi.fn(),
            close: vi.fn(),
            add: vi.fn(),
        });

        const { loadSchema } = await import("./schemaLoader.js");
        vi.mocked(loadSchema).mockReturnValue(mockResourceManageSchema);

        const fs = await import("fs");
        vi.mocked(fs.readFileSync).mockReturnValue("{\"test\": \"schema\"}");

        const path = await import("path");
        vi.mocked(path.join).mockImplementation((...args) => args.join("/"));
        vi.mocked(path.dirname).mockImplementation((p) => p.split("/").slice(0, -1).join("/"));

        const url = await import("node:url");
        vi.mocked(url.fileURLToPath).mockImplementation((u) => u.replace("file://", ""));

        // Set up fresh tool instances
        tools = new BuiltInTools(testUser);
        adminTools = new BuiltInTools(adminUser);

        // Store mocked imports for test access
        mockProcessRun = await import("../../tasks/run/queue.js");
        mockProcessNewSwarmExecution = await import("../../tasks/swarm/queue.js");
        mockActiveSwarmRegistry = await import("../../tasks/swarm/process.js");
    });

    afterEach(async () => {
        // Validate cleanup after each test
        // Note: We expect 2 user records (testUser and adminUser) to exist throughout the test suite
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["chat", "chat_message", "resource", "resource_version", "team"],
            logOrphans: false,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans.length, "orphans");
        }
    });

    describe("DefineTool", () => {
        it("should return schema for ResourceManage find operation", async () => {
            const result = await tools.defineTool(defineToolFixtures.resourceManageFind);

            expect(result.isError).toBe(false);
            expect(result.content[0].type).toBe("text");

            const schema = JSON.parse(result.content[0].text);
            expect(schema.properties.resource_type.const).toBe("Note");
            expect(schema.properties.filters).toBeDefined();
            expect(schema.description).toContain("find a resource of type Note");
        });

        it("should return schema for ResourceManage add operation", async () => {
            const result = await tools.defineTool(defineToolFixtures.resourceManageAdd);

            expect(result.isError).toBe(false);
            const schema = JSON.parse(result.content[0].text);
            expect(schema.properties.resource_type.const).toBe("Project");
            expect(schema.properties.attributes).toBeDefined();
            expect(schema.description).toContain("add a resource of type Project");
        });

        it("should return schema for ResourceManage update operation", async () => {
            const result = await tools.defineTool(defineToolFixtures.resourceManageUpdate);

            expect(result.isError).toBe(false);
            const schema = JSON.parse(result.content[0].text);
            expect(schema.properties.resource_type.const).toBe("Team");
            expect(schema.properties.attributes).toBeDefined();
            expect(schema.description).toContain("update a resource of type Team");
        });

        it("should return schema for ResourceManage delete operation", async () => {
            const result = await tools.defineTool(defineToolFixtures.resourceManageDelete);

            expect(result.isError).toBe(false);
            const schema = JSON.parse(result.content[0].text);
            expect(schema.properties.op.const).toBe("delete");
            expect(schema.description).toContain("delete a resource of type Bot");
        });

        it("should return error for unsupported tool", async () => {
            const result = await tools.defineTool(defineToolFixtures.unsupportedTool);

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("Error: defineTool only supports ResourceManage");
        });

        it("should return error for invalid operation", async () => {
            const result = await tools.defineTool({
                toolName: McpToolName.ResourceManage,
                variant: "Note",
                op: "invalid_op" as any,
            });

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("Could not find schema for operation");
        });
    });

    describe("SendMessage", () => {
        let testChatId: string;

        beforeEach(async () => {
            // Create test chat
            const chat = await DbProvider.get().chat.create({
                data: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    isPrivate: false,
                    openToAnyoneWithInvite: true,
                    config: {},
                    createdById: testUser.id,
                },
            });
            testChatId = chat.id.toString();
        });

        afterEach(async () => {
            // Clean up test chat and related data
            await DbProvider.get().chat_message.deleteMany();
            await DbProvider.get().chat_participants.deleteMany();
            await DbProvider.get().chat.deleteMany({ where: { id: BigInt(testChatId) } });
        });

        it("should send message to chat successfully", async () => {
            const params = {
                ...sendMessageFixtures.toChatId,
                recipient: { kind: "chat" as const, chatId: testChatId },
            };

            const result = await tools.sendMessage(params);

            expect(result.isError).toBe(false);
            expect(result.content[0].text).toContain("Message sent to chat");
            expect(result.content[0].text).toContain(testChatId);
        });

        it("should handle complex content objects", async () => {
            const params = {
                ...sendMessageFixtures.withComplexContent,
                recipient: { kind: "chat" as const, chatId: testChatId },
            };

            const result = await tools.sendMessage(params);

            expect(result.isError).toBe(false);
            expect(result.content[0].text).toContain("Message sent to chat");
        });

        it("should handle missing metadata gracefully", async () => {
            const params = {
                recipient: { kind: "chat" as const, chatId: testChatId },
                content: "Simple message without metadata",
            };

            const result = await tools.sendMessage(params);

            expect(result.isError).toBe(false);
            expect(result.content[0].text).toContain("Message sent to chat");
        });

        it("should return error for bot recipient (not implemented)", async () => {
            const result = await tools.sendMessage(sendMessageFixtures.toBotId);

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("Recipient kind 'bot' resolution to a chat ID is not yet implemented");
        });

        it("should return error for user recipient (not implemented)", async () => {
            const result = await tools.sendMessage(sendMessageFixtures.toUserId);

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("Recipient kind 'user' resolution to a chat ID is not yet implemented");
        });

        it("should return error for topic recipient (not implemented)", async () => {
            const result = await tools.sendMessage(sendMessageFixtures.toTopic);

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("Recipient kind 'topic' is not yet implemented");
        });

        it("should return error for invalid recipient", async () => {
            const result = await tools.sendMessage(sendMessageFixtures.invalidRecipient);

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("Unknown recipient kind");
        });

        it("should handle database errors gracefully", async () => {
            // Mock database error
            const mockCreateOneHelper = vi.fn().mockRejectedValue(new Error("Database connection failed"));
            vi.doMock("../../actions/creates.js", () => ({
                createOneHelper: mockCreateOneHelper,
            }));

            const params = {
                ...sendMessageFixtures.toChatId,
                recipient: { kind: "chat" as const, chatId: testChatId },
            };

            const result = await tools.sendMessage(params);

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("Database connection failed");
        });
    });

    describe("ResourceManage", () => {
        let testResourceId: string;
        let testTeamId: string;

        beforeEach(async () => {
            // Create test resource
            const resource = await DbProvider.get().resource.create({
                data: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    resourceType: ResourceType.Note,
                    isPrivate: false,
                    createdById: testUser.id,
                    versions: {
                        create: {
                            id: generatePK(),
                            publicId: generatePK().toString(),
                            versionLabel: "1.0.0",
                            isComplete: true,
                            createdById: testUser.id,
                            translations: {
                                create: {
                                    id: generatePK(),
                                    language: DEFAULT_LANGUAGE,
                                    name: "Test Note",
                                    description: "Test description",
                                },
                            },
                        },
                    },
                },
                include: {
                    versions: {
                        include: {
                            translations: true,
                        },
                    },
                },
            });
            testResourceId = resource.versions[0].id.toString();

            // Create test team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    handle: "test-team",
                    isPrivate: false,
                    config: {},
                    createdById: testUser.id,
                },
            });
            testTeamId = team.id.toString();
        });

        afterEach(async () => {
            // Clean up test resources and teams in dependency order
            await DbProvider.get().resource_translation.deleteMany();
            await DbProvider.get().resource_version.deleteMany();
            await DbProvider.get().resource.deleteMany();
            await DbProvider.get().member.deleteMany();
            await DbProvider.get().team_translation.deleteMany();
            await DbProvider.get().team.deleteMany();
        });

        describe("Find operations", () => {
            it("should find notes successfully", async () => {
                const result = await tools.resourceManage(resourceManageFixtures.findNotes);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
                // The actual structure depends on readManyHelper implementation
            });

            it("should find projects successfully", async () => {
                const result = await tools.resourceManage(resourceManageFixtures.findProjects);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });

            it("should handle find with filters", async () => {
                const params = {
                    ...resourceManageFixtures.findNotes,
                    filters: {
                        isPrivate: false,
                        createdTimeFrame: { after: "2024-01-01" },
                    },
                };

                const result = await tools.resourceManage(params);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });
        });

        describe("Add operations", () => {
            it("should create note successfully", async () => {
                const result = await tools.resourceManage(resourceManageFixtures.addNote);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });

            it("should create project successfully", async () => {
                const result = await tools.resourceManage(resourceManageFixtures.addProject);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });

            it("should create team successfully", async () => {
                const result = await tools.resourceManage(resourceManageFixtures.addTeam);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });

            it("should create bot successfully", async () => {
                const result = await tools.resourceManage(resourceManageFixtures.addBot);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });

            it("should handle invalid attributes for note", async () => {
                const params = {
                    op: "add" as const,
                    resource_type: "Note",
                    attributes: {
                        // Missing required 'name' field
                        content: "Content without name",
                    },
                };

                const result = await tools.resourceManage(params);

                expect(result.isError).toBe(true);
                expect(result.content[0].text).toContain("Invalid attributes for Note");
            });
        });

        describe("Update operations", () => {
            it("should update note successfully", async () => {
                const params = {
                    ...resourceManageFixtures.updateNote,
                    id: testResourceId,
                };

                const result = await tools.resourceManage(params);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });

            it("should update team successfully", async () => {
                const params = {
                    ...resourceManageFixtures.updateTeam,
                    id: testTeamId,
                };

                const result = await tools.resourceManage(params);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });

            it("should handle invalid resource ID", async () => {
                const params = {
                    ...resourceManageFixtures.updateNote,
                    id: "invalid-id",
                };

                const result = await tools.resourceManage(params);

                expect(result.isError).toBe(true);
                // Error message depends on updateOneHelper implementation
            });
        });

        describe("Delete operations", () => {
            it("should delete resource successfully", async () => {
                const params = {
                    ...resourceManageFixtures.deleteResource,
                    id: testResourceId,
                };

                const result = await tools.resourceManage(params);

                expect(result.isError).toBe(false);
                const data = JSON.parse(result.content[0].text);
                expect(data).toBeDefined();
            });

            it("should handle delete of non-existent resource", async () => {
                const params = {
                    ...resourceManageFixtures.deleteResource,
                    id: "non-existent-id",
                };

                const result = await tools.resourceManage(params);

                expect(result.isError).toBe(true);
                // Error message depends on deleteOneHelper implementation
            });
        });

        describe("Invalid operations", () => {
            it("should handle invalid operation", async () => {
                const result = await tools.resourceManage(resourceManageFixtures.invalidOperation);

                expect(result.isError).toBe(true);
                expect(result.content[0].text).toContain("Unhandled resource manage operation");
            });
        });
    });

    describe("RunRoutine", () => {
        describe("Start routine operations", () => {
            it("should start routine in standalone mode", async () => {
                const result = await tools.runRoutine(runRoutineFixtures.startBasic);

                expect(result.isError).toBe(false);
                expect(result.content[0].text).toContain("Run started successfully");
                expect(mockProcessRun.processRun).toHaveBeenCalled();
            });

            it("should start routine with inputs", async () => {
                const result = await tools.runRoutine(runRoutineFixtures.startWithInputs);

                expect(result.isError).toBe(false);
                expect(result.content[0].text).toContain("Run scheduled successfully");
                expect(mockProcessRun.processRun).toHaveBeenCalled();

                // Verify inputs were passed correctly
                const callArgs = mockProcessRun.processRun.mock.calls[0][0];
                expect(callArgs.input.formValues).toEqual(runRoutineFixtures.startWithInputs.inputs);
                expect(callArgs.input.config.isTimeSensitive).toBe(true); // priority: "high"
            });

            it("should start routine in swarm-integrated mode", async () => {
                // Mock active swarm
                const mockSwarmId = generatePK().toString();
                mockActiveSwarmRegistry.activeSwarmRegistry.getOrderedRecords.mockReturnValue([
                    { id: mockSwarmId, createdAt: new Date(), userId: testUser.id },
                ]);
                mockActiveSwarmRegistry.activeSwarmRegistry.get.mockReturnValue({
                    getAssociatedUserId: () => testUser.id,
                });

                const result = await tools.runRoutine(runRoutineFixtures.startBasic);

                expect(result.isError).toBe(false);
                expect(result.content[0].text).toContain("swarm-integrated mode");
                expect(result.content[0].text).toContain(mockSwarmId);
                expect(mockProcessRun.processRun).toHaveBeenCalled();

                // Verify swarm context was passed
                const callArgs = mockProcessRun.processRun.mock.calls[0][0];
                expect(callArgs.context.swarmId).toBe(mockSwarmId);
                expect(callArgs.context.parentSwarmId).toBe(mockSwarmId);
            });

            it("should handle queue processing failure", async () => {
                mockProcessRun.processRun.mockResolvedValue(mockQueueResults.runFailure);

                const result = await tools.runRoutine(runRoutineFixtures.startBasic);

                expect(result.isError).toBe(true);
                expect(result.content[0].text).toContain("Failed to queue run task");
            });
        });

        describe("Manage routine operations", () => {
            let mockStateMachine: any;

            beforeEach(() => {
                mockStateMachine = {
                    getState: vi.fn().mockReturnValue("running"),
                    pause: vi.fn().mockResolvedValue(undefined),
                    stop: vi.fn().mockResolvedValue(undefined),
                };

                // Mock activeRunRegistry
                vi.doMock("../../tasks/run/process.js", () => ({
                    activeRunRegistry: {
                        get: vi.fn().mockReturnValue(mockStateMachine),
                    },
                }));
            });

            it("should get run status successfully", async () => {
                const result = await tools.runRoutine(runRoutineFixtures.statusRun);

                expect(result.isError).toBe(false);
                expect(result.content[0].text).toContain("current status: running");
            });

            it("should pause run successfully", async () => {
                const result = await tools.runRoutine(runRoutineFixtures.pauseRun);

                expect(result.isError).toBe(false);
                expect(result.content[0].text).toContain("pause request initiated successfully");
                expect(mockStateMachine.pause).toHaveBeenCalled();
            });

            it("should handle pause failure", async () => {
                mockStateMachine.pause.mockRejectedValue(new Error("Cannot pause - already stopped"));

                const result = await tools.runRoutine(runRoutineFixtures.pauseRun);

                expect(result.isError).toBe(true);
                expect(result.content[0].text).toContain("Failed to pause run");
                expect(result.content[0].text).toContain("Cannot pause - already stopped");
            });

            it("should cancel run successfully", async () => {
                const result = await tools.runRoutine(runRoutineFixtures.cancelRun);

                expect(result.isError).toBe(false);
                expect(result.content[0].text).toContain("cancellation request initiated successfully");
                expect(mockStateMachine.stop).toHaveBeenCalledWith({
                    reason: "Cancelled by user via MCP tool",
                });
            });

            it("should handle run not found", async () => {
                vi.doMock("../../tasks/run/process.js", () => ({
                    activeRunRegistry: {
                        get: vi.fn().mockReturnValue(null),
                    },
                }));

                const result = await tools.runRoutine(runRoutineFixtures.statusRun);

                expect(result.isError).toBe(true);
                expect(result.content[0].text).toContain("Run");
                expect(result.content[0].text).toContain("not found in active runs registry");
            });

            it("should handle resume operation (not implemented)", async () => {
                const result = await tools.runRoutine(runRoutineFixtures.resumeRun);

                expect(result.isError).toBe(true);
                expect(result.content[0].text).toContain("Failed to resume run");
                expect(result.content[0].text).toContain("not implemented");
            });
        });

        describe("Error handling", () => {
            it("should handle invalid routine ID", async () => {
                const params = {
                    ...runRoutineFixtures.startBasic,
                    routineId: "invalid-routine-id",
                };

                // Mock processRun to throw error
                mockProcessRun.processRun.mockRejectedValue(new Error("Routine not found"));

                const result = await tools.runRoutine(params);

                expect(result.isError).toBe(true);
                expect(result.content[0].text).toContain("Failed to start routine");
                expect(result.content[0].text).toContain("Routine not found");
            });
        });
    });

    describe("SpawnSwarm", () => {
        beforeEach(() => {
            // Mock active parent swarm
            const mockParentSwarmId = generatePK().toString();
            mockActiveSwarmRegistry.activeSwarmRegistry.getOrderedRecords.mockReturnValue([
                { id: mockParentSwarmId, createdAt: new Date(), userId: testUser.id },
            ]);
            mockActiveSwarmRegistry.activeSwarmRegistry.get.mockReturnValue({
                getAssociatedUserId: () => testUser.id,
            });
        });

        it("should spawn simple swarm successfully", async () => {
            const result = await tools.spawnSwarm(spawnSwarmFixtures.simpleSwarm);

            expect(result.isError).toBe(false);
            expect(result.content[0].text).toContain("Child swarm");
            expect(result.content[0].text).toContain("spawned successfully");
            expect(result.content[0].text).toContain(spawnSwarmFixtures.simpleSwarm.goal);
            expect(mockProcessNewSwarmExecution.processNewSwarmExecution).toHaveBeenCalled();
        });

        it("should spawn rich swarm successfully", async () => {
            const result = await tools.spawnSwarm(spawnSwarmFixtures.richSwarm);

            expect(result.isError).toBe(false);
            expect(result.content[0].text).toContain("Rich child swarm");
            expect(result.content[0].text).toContain("spawned successfully");
            expect(result.content[0].text).toContain(spawnSwarmFixtures.richSwarm.teamId);
            expect(mockProcessNewSwarmExecution.processNewSwarmExecution).toHaveBeenCalled();

            // Verify rich swarm gets better model
            const callArgs = mockProcessNewSwarmExecution.processNewSwarmExecution.mock.calls[0][0];
            expect(callArgs.input.executionConfig.model).toBe("claude-3-sonnet-20240229");
            expect(callArgs.input.executionConfig.parallelExecutionLimit).toBe(5);
        });

        it("should handle resource allocation correctly", async () => {
            const result = await tools.spawnSwarm(spawnSwarmFixtures.simpleSwarm);

            expect(result.isError).toBe(false);
            expect(result.content[0].text).toContain("Reserved resources");

            const callArgs = mockProcessNewSwarmExecution.processNewSwarmExecution.mock.calls[0][0];
            expect(callArgs.allocation).toBeDefined();
            expect(callArgs.allocation.maxCredits).toBeDefined();
            expect(callArgs.allocation.maxDurationMs).toBeDefined();
        });

        it("should fail when no parent swarm context", async () => {
            // Clear parent swarm context
            mockActiveSwarmRegistry.activeSwarmRegistry.getOrderedRecords.mockReturnValue([]);
            mockActiveSwarmRegistry.activeSwarmRegistry.get.mockReturnValue(null);

            const result = await tools.spawnSwarm(spawnSwarmFixtures.simpleSwarm);

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("can only be called from within an active swarm context");
        });

        it("should handle queue processing failure", async () => {
            mockProcessNewSwarmExecution.processNewSwarmExecution.mockRejectedValue(
                new Error("Failed to spawn swarm"),
            );

            const result = await tools.spawnSwarm(spawnSwarmFixtures.simpleSwarm);

            expect(result.isError).toBe(true);
            expect(result.content[0].text).toContain("Failed to spawn child swarm");
            expect(result.content[0].text).toContain("Failed to spawn swarm");
        });

        it("should use provided parent swarm ID", async () => {
            const parentSwarmId = generatePK().toString();

            const result = await tools.spawnSwarm(spawnSwarmFixtures.simpleSwarm, parentSwarmId);

            expect(result.isError).toBe(false);
            const callArgs = mockProcessNewSwarmExecution.processNewSwarmExecution.mock.calls[0][0];
            expect(callArgs.context.parentSwarmId).toBe(parentSwarmId);
        });
    });

    describe("Type guards and validation", () => {
        it("should validate recipient types correctly", async () => {
            const validRecipients = [
                { kind: "chat", chatId: generatePK().toString() },
                { kind: "bot", botId: generatePK().toString() },
                { kind: "user", userId: generatePK().toString() },
                { kind: "topic", topic: "test-topic" },
            ];

            for (const recipient of validRecipients) {
                const params = {
                    recipient,
                    content: "Test message",
                };

                const result = await tools.sendMessage(params);

                if (recipient.kind === "chat") {
                    // Chat should fail because we didn't create a real chat
                    expect(result.isError).toBe(true);
                } else {
                    // Others should fail with "not implemented" messages
                    expect(result.isError).toBe(true);
                    expect(result.content[0].text).toContain("not yet implemented");
                }
            }
        });

        it("should validate resource attributes correctly", async () => {
            // Test various invalid attribute combinations
            const invalidAttributes = [
                {
                    op: "add",
                    resource_type: "Note",
                    attributes: {}, // Missing required name
                },
                {
                    op: "add",
                    resource_type: "Project",
                    attributes: {}, // Missing required name
                },
                {
                    op: "update",
                    id: generatePK().toString(),
                    resource_type: "Note",
                    attributes: {
                        name: 123, // Wrong type
                    },
                },
            ];

            for (const params of invalidAttributes) {
                const result = await tools.resourceManage(params as any);
                expect(result.isError).toBe(true);
                expect(result.content[0].text).toContain("Invalid attributes");
            }
        });
    });

    describe("Private methods and helpers", () => {
        it("should map find inputs correctly", () => {
            const result = (tools as any)._mapFindToInput("Note", { isPrivate: false });

            expect(result.rootResourceType).toBe(ResourceType.Note);
            expect(result.isPrivate).toBe(false);
        });

        it("should map routine subtypes correctly", () => {
            const result = (tools as any)._mapFindToInput("RoutineMultiStep", {});

            expect(result.rootResourceType).toBe(ResourceType.Routine);
            expect(result.resourceSubType).toBe(ResourceSubType.RoutineMultiStep);
        });

        it("should map standard subtypes correctly", () => {
            const result = (tools as any)._mapFindToInput("StandardDataStructure", {});

            expect(result.rootResourceType).toBe(ResourceType.Standard);
            expect(result.resourceSubType).toBe(ResourceSubType.StandardDataStructure);
        });

        it("should create resource version inputs correctly", () => {
            const result = (tools as any)._createResourceVersionInput(
                ResourceType.Note,
                undefined,
                [{ id: "1", language: "en", name: "Test" }],
                { isPrivate: true, rootCreate: { isPrivate: true } },
            );

            expect(result.rootCreate.resourceType).toBe(ResourceType.Note);
            expect(result.isPrivate).toBe(true);
            expect(result.translationsCreate).toHaveLength(1);
        });
    });
});

describe("SwarmTools", () => {
    let swarmTools: SwarmTools;
    let testUser: SessionUser;

    beforeAll(async () => {
        const users = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
        testUser = {
            ...users.records[0],
            languages: [DEFAULT_LANGUAGE],
            isLoggedIn: true,
            roles: [],
            wallets: [],
            walletIdCurrent: null,
            isAdmin: false,
            theme: "light",
        };
    });

    afterAll(async () => {
        await cleanupGroups.userAuth(DbProvider.get());
        await cleanupGroups.team(DbProvider.get());
    });

    beforeEach(() => {
        vi.clearAllMocks();
        swarmTools = new SwarmTools(mockConversationStore);

        // Reset conversation store mocks
        mockConversationStore.get.mockResolvedValue(mockConversationState);
        mockConversationStore.updateTeamConfig.mockResolvedValue(undefined);
    });

    describe("updateSwarmSharedState", () => {
        it("should update subtasks successfully", async () => {
            const result = await swarmTools.updateSwarmSharedState(
                mockConversationState.conversationId,
                swarmToolsFixtures.updateSubTasks,
                testUser,
            );

            expect(result.success).toBe(true);
            expect(result.updatedSubTasks).toBeDefined();
            expect(result.updatedSubTasks?.length).toBeGreaterThan(0);
        });

        it("should update blackboard successfully", async () => {
            const result = await swarmTools.updateSwarmSharedState(
                mockConversationState.conversationId,
                swarmToolsFixtures.updateBlackboard,
                testUser,
            );

            expect(result.success).toBe(true);
            expect(result.updatedSharedScratchpad).toBeDefined();
            expect(result.updatedSharedScratchpad?.analysis_results).toBeDefined();
        });

        it("should update team config successfully", async () => {
            const result = await swarmTools.updateSwarmSharedState(
                mockConversationState.conversationId,
                swarmToolsFixtures.updateTeamConfig,
                testUser,
            );

            expect(result.success).toBe(true);
            expect(result.updatedTeamConfig).toBeDefined();
        });

        it("should handle conversation not found", async () => {
            mockConversationStore.get.mockResolvedValue(null);

            const result = await swarmTools.updateSwarmSharedState(
                "non-existent-conversation",
                swarmToolsFixtures.updateSubTasks,
                testUser,
            );

            expect(result.success).toBe(false);
            expect(result.error).toBe("CONVERSATION_STATE_NOT_FOUND");
        });

        it("should handle team config update without user session", async () => {
            const result = await swarmTools.updateSwarmSharedState(
                mockConversationState.conversationId,
                swarmToolsFixtures.updateTeamConfig,
                // No user provided
            );

            expect(result.success).toBe(false);
            expect(result.error).toBe("SESSION_USER_REQUIRED");
        });

        it("should handle team not found", async () => {
            // Mock team read failure
            vi.doMock("../../actions/reads.js", () => ({
                readOneHelper: vi.fn().mockResolvedValue(null),
            }));

            const result = await swarmTools.updateSwarmSharedState(
                mockConversationState.conversationId,
                swarmToolsFixtures.updateTeamConfig,
                testUser,
            );

            expect(result.success).toBe(false);
            expect(result.error).toBe("TEAM_NOT_FOUND_OR_ACCESS_DENIED");
        });
    });

    describe("endSwarm", () => {
        beforeEach(() => {
            // Mock active swarm registry
            mockActiveSwarmRegistry.activeSwarmRegistry.get.mockReturnValue({
                getAssociatedUserId: () => testUser.id,
                stop: vi.fn().mockResolvedValue({
                    success: true,
                    message: "Swarm ended successfully",
                    finalState: {
                        endedAt: new Date().toISOString(),
                        reason: "User requested",
                        mode: "graceful",
                        totalSubTasks: 3,
                        completedSubTasks: 2,
                        totalCreditsUsed: "150",
                        totalToolCalls: 12,
                    },
                }),
            });
        });

        it("should end swarm gracefully", async () => {
            const result = await swarmTools.endSwarm(
                mockConversationState.conversationId,
                swarmToolsFixtures.endSwarmGraceful,
                testUser,
            );

            expect(result.success).toBe(true);
            expect(result.finalState?.mode).toBe("graceful");
        });

        it("should force end swarm", async () => {
            const result = await swarmTools.endSwarm(
                mockConversationState.conversationId,
                swarmToolsFixtures.endSwarmForce,
                testUser,
            );

            expect(result.success).toBe(true);
            expect(result.finalState?.mode).toBe("graceful"); // Mocked value
        });

        it("should handle unauthorized user", async () => {
            // Mock different user ID
            mockActiveSwarmRegistry.activeSwarmRegistry.get.mockReturnValue({
                getAssociatedUserId: () => "different-user-id",
                stop: vi.fn(),
            });

            const result = await swarmTools.endSwarm(
                mockConversationState.conversationId,
                swarmToolsFixtures.endSwarmGraceful,
                testUser,
            );

            expect(result.success).toBe(false);
            expect(result.error).toBe("INSUFFICIENT_PERMISSIONS");
        });

        it("should allow admin to end any swarm", async () => {
            // Mock different user but test user is admin
            const adminUser = { ...testUser, isAdmin: true };
            mockActiveSwarmRegistry.activeSwarmRegistry.get.mockReturnValue({
                getAssociatedUserId: () => "different-user-id",
                stop: vi.fn().mockResolvedValue({ success: true }),
            });

            // Mock admin ID check
            vi.doMock("../../db/provider.js", () => ({
                DbProvider: {
                    getAdminId: vi.fn().mockResolvedValue(adminUser.id),
                },
            }));

            const result = await swarmTools.endSwarm(
                mockConversationState.conversationId,
                swarmToolsFixtures.endSwarmGraceful,
                adminUser,
            );

            expect(result.success).toBe(true);
        });

        it("should handle missing authentication", async () => {
            const result = await swarmTools.endSwarm(
                mockConversationState.conversationId,
                swarmToolsFixtures.endSwarmGraceful,
                { ...testUser, id: "" }, // Invalid user
            );

            expect(result.success).toBe(false);
            expect(result.error).toBe("AUTHENTICATION_REQUIRED");
        });

        it("should handle no active swarm instance", async () => {
            mockActiveSwarmRegistry.activeSwarmRegistry.get.mockReturnValue(null);

            const result = await swarmTools.endSwarm(
                mockConversationState.conversationId,
                swarmToolsFixtures.endSwarmGraceful,
                testUser,
            );

            expect(result.success).toBe(false);
            expect(result.error).toBe("NO_ACTIVE_SWARM_INSTANCE");
        });

        it("should handle swarm stop failure", async () => {
            mockActiveSwarmRegistry.activeSwarmRegistry.get.mockReturnValue({
                getAssociatedUserId: () => testUser.id,
                stop: vi.fn().mockRejectedValue(new Error("Stop failed")),
            });

            const result = await swarmTools.endSwarm(
                mockConversationState.conversationId,
                swarmToolsFixtures.endSwarmGraceful,
                testUser,
            );

            expect(result.success).toBe(false);
            expect(result.error).toBe("SWARM_STATE_MACHINE_ERROR");
        });
    });

    describe("Error handling and edge cases", () => {
        it("should handle malformed conversation state", async () => {
            mockConversationStore.get.mockResolvedValue({
                conversationId: mockConversationState.conversationId,
                userId: testUser.id,
                config: null, // Invalid config
            });

            const result = await swarmTools.updateSwarmSharedState(
                mockConversationState.conversationId,
                swarmToolsFixtures.updateSubTasks,
                testUser,
            );

            expect(result.success).toBe(false);
            expect(result.error).toBe("CONVERSATION_CONFIG_NOT_FOUND");
        });

        it("should handle database errors gracefully", async () => {
            mockConversationStore.get.mockRejectedValue(new Error("Database connection failed"));

            const result = await swarmTools.updateSwarmSharedState(
                mockConversationState.conversationId,
                swarmToolsFixtures.updateSubTasks,
                testUser,
            );

            expect(result.success).toBe(false);
        });
    });
});
