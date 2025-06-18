import { generatePK, generatePublicId, RunStatus } from "@vrooli/shared";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { MockedFunction } from "vitest";
import { generateEmbeddings } from "./embeddings.js";

import { DbProvider } from "@vrooli/server";

// Mock the EmbeddingService
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        EmbeddingService: {
            get: () => ({
                getEmbeddings: vi.fn().mockImplementation((objectType: string, sentences: string[]) => {
                    console.log(`Mock getEmbeddings called for ${objectType} with ${sentences.length} sentences`);
                    // Return mock embeddings - arrays of 1536 dimensions (OpenAI standard)
                    return sentences.map(() => Array(1536).fill(0.1));
                }),
            }),
            getEmbeddableString: vi.fn().mockImplementation((data: string | Record<string, any>, language?: string) => {
                // Simple mock implementation
                const parts = [];
                if (data.name) parts.push(data.name);
                if (data.handle) parts.push(data.handle);
                if (data.description) parts.push(data.description);
                if (data.bio) parts.push(data.bio);
                return parts.join(" ");
            }),
        },
    };
});

describe("generateEmbeddings integration tests", () => {
    beforeAll(() => {
        // Suppress logger output during tests
        vi.spyOn(console, "error").mockImplementation(() => {});
        vi.spyOn(console, "log").mockImplementation(() => {});
        vi.spyOn(console, "warn").mockImplementation(() => {});
    });

    // Helper function to validate embeddings since Prisma can't deserialize vector types
    async function validateEmbedding(tableName: string, id: bigint, shouldHaveEmbedding: boolean = true) {
        const result = await DbProvider.get().$queryRawUnsafe<Array<{
            id: bigint;
            has_embedding: boolean;
            embeddingExpiredAt: Date | null;
        }>>(
            `SELECT id, (embedding IS NOT NULL) as has_embedding, "embeddingExpiredAt" FROM "${tableName}" WHERE id = ${id}`
        );

        expect(result).toHaveLength(1);
        expect(result[0].has_embedding).toBe(shouldHaveEmbedding);
        if (shouldHaveEmbedding) {
            expect(result[0].embeddingExpiredAt).toBeDefined();
            expect(result[0].embeddingExpiredAt!.getTime()).toBeGreaterThan(Date.now() - 5000); // Allow 5 seconds tolerance
        }
    }
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testChatIds: bigint[] = [];
    const testIssueIds: bigint[] = [];
    const testMeetingIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testResourceVersionIds: bigint[] = [];
    const testTagIds: bigint[] = [];
    const testReminderIds: bigint[] = [];
    const testRunIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testChatIds.length = 0;
        testIssueIds.length = 0;
        testMeetingIds.length = 0;
        testResourceIds.length = 0;
        testResourceVersionIds.length = 0;
        testTagIds.length = 0;
        testReminderIds.length = 0;
        testRunIds.length = 0;
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testRunIds.length > 0) {
            await db.run.deleteMany({ where: { id: { in: testRunIds } } });
        }
        if (testReminderIds.length > 0) {
            await db.reminder.deleteMany({ where: { id: { in: testReminderIds } } });
        }
        if (testResourceVersionIds.length > 0) {
            await db.resource_version.deleteMany({ where: { id: { in: testResourceVersionIds } } });
        }
        if (testResourceIds.length > 0) {
            await db.resource.deleteMany({ where: { id: { in: testResourceIds } } });
        }
        if (testTagIds.length > 0) {
            await db.tag.deleteMany({ where: { id: { in: testTagIds } } });
        }
        if (testMeetingIds.length > 0) {
            await db.meeting.deleteMany({ where: { id: { in: testMeetingIds } } });
        }
        if (testIssueIds.length > 0) {
            await db.issue.deleteMany({ where: { id: { in: testIssueIds } } });
        }
        if (testChatIds.length > 0) {
            await db.chat.deleteMany({ where: { id: { in: testChatIds } } });
        }
        if (testTeamIds.length > 0) {
            await db.team.deleteMany({ where: { id: { in: testTeamIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should generate embeddings for users with expired or missing embeddings", async () => {
        // Create a user with translations that need embeddings
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User",
                handle: "testuser",
                isBot: false,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        bio: "A test user bio",
                        embeddingExpiredAt: new Date(Date.now() - 1000), // Expired
                    }],
                },
            },
            include: {
                translations: true,
            },
        });
        testUserIds.push(user.id);

        // Run the embeddings generation
        try {
            await generateEmbeddings();
        } catch (error) {
            console.error("generateEmbeddings error:", error);
            throw error;
        }

        // Check that embeddings were updated
        await validateEmbedding("user_translation", user.translations[0].id);
    });

    it("should generate embeddings for teams with bio translations", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner",
                isBot: false,
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id }
                },
                handle: "testteam",
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Team",
                        bio: "We are a test team",
                        embeddingExpiredAt: null, // No embedding yet
                    }],
                },
            },
            include: {
                translations: true,
            },
        });
        testTeamIds.push(team.id);

        await generateEmbeddings();

        // Check that embeddings were updated
        await validateEmbedding("team_translation", team.translations[0].id);
    });

    it("should generate embeddings for chats with name and description", async () => {
        const creator = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Chat Creator",
                handle: "chatcreator",
                isBot: false,
            },
        });
        testUserIds.push(creator.id);

        const chat = await DbProvider.get().chat.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                creatorId: creator.id,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Chat",
                        description: "A chat for testing embeddings",
                        embeddingExpiredAt: new Date(0), // Very old, needs update
                    }],
                },
            },
            include: {
                translations: true,
            },
        });
        testChatIds.push(chat.id);

        await generateEmbeddings();

        // Check that embeddings were updated
        await validateEmbedding("chat_translation", chat.translations[0].id);
    });

    it("should generate embeddings for issues", async () => {
        const creator = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Issue Creator",
                handle: "issuecreator",
                isBot: false,
            },
        });
        testUserIds.push(creator.id);

        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: creator.id }
                },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Issue",
                        description: "This is a test issue",
                        embeddingExpiredAt: null,
                    }],
                },
            },
            include: {
                translations: true,
            },
        });
        testIssueIds.push(issue.id);

        await generateEmbeddings();

        // Check that embeddings were updated
        await validateEmbedding("issue_translation", issue.translations[0].id);
    });

    it("should generate embeddings for meetings", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Meeting Owner",
                handle: "meetingowner",
                isBot: false,
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id }
                },
                handle: "meetingteam",
            },
        });
        testTeamIds.push(team.id);

        const meeting = await DbProvider.get().meeting.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                teamId: team.id,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Meeting",
                        description: "Quarterly planning meeting",
                        embeddingExpiredAt: new Date(Date.now() - 86400000), // Yesterday
                    }],
                },
            },
            include: {
                translations: true,
            },
        });
        testMeetingIds.push(meeting.id);

        await generateEmbeddings();

        // Check that embeddings were updated
        await validateEmbedding("meeting_translation", meeting.translations[0].id);
    });

    it("should generate embeddings for resource versions", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner",
                isBot: false,
            },
        });
        testUserIds.push(owner.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id }
                },
                resourceType: "Routine",
                isPrivate: false,
                isInternal: false,
            },
        });
        testResourceIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionLabel: "1.0.0",
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Resource",
                        description: "A test resource version",
                        embeddingExpiredAt: null,
                    }],
                },
            },
            include: {
                translations: true,
            },
        });
        testResourceVersionIds.push(resourceVersion.id);

        await generateEmbeddings();

        // Check that embeddings were updated
        await validateEmbedding("resource_translation", resourceVersion.translations[0].id);
    });

    it("should generate embeddings for tags", async () => {
        const creator = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Tag Creator",
                handle: "tagcreator",
                isBot: false,
            },
        });
        testUserIds.push(creator.id);

        const tag = await DbProvider.get().tag.create({
            data: {
                id: generatePK(),
                createdBy: {
                    connect: { id: creator.id }
                },
                tag: "test-tag",
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        description: "A tag for testing purposes",
                        embeddingExpiredAt: null,
                    }],
                },
            },
            include: {
                translations: true,
            },
        });
        testTagIds.push(tag.id);

        await generateEmbeddings();

        // Check that embeddings were updated
        await validateEmbedding("tag_translation", tag.translations[0].id);
    });

    it("should generate embeddings for reminders (non-translatable)", async () => {
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reminder User",
                handle: "reminderuser",
                isBot: false,
            },
        });
        testUserIds.push(user.id);

        // Create reminder list first
        const reminderList = await DbProvider.get().reminder_list.create({
            data: {
                id: generatePK(),
                userId: user.id,
            },
        });

        const reminder = await DbProvider.get().reminder.create({
            data: {
                id: generatePK(),
                reminderListId: reminderList.id,
                name: "Test Reminder",
                description: "Remember to test embeddings",
                dueDate: new Date(Date.now() + 86400000), // Tomorrow
                index: 0,
                embeddingExpiredAt: null,
            },
        });
        testReminderIds.push(reminder.id);

        await generateEmbeddings();

        // Check that embeddings were updated
        await validateEmbedding("reminder", reminder.id);
    });

    // Skipping run embeddings test - Run model doesn't have embedding fields in schema

    it("should skip entities with current embeddings", async () => {
        const futureDate = new Date(Date.now() + 86400000); // Tomorrow
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "User with Current Embedding",
                handle: "currentembedding",
                isBot: false,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        bio: "Already has embedding",
                        embeddingExpiredAt: futureDate, // Not expired
                    }],
                },
            },
        });
        testUserIds.push(user.id);

        // Clear mock calls
        vi.clearAllMocks();

        await generateEmbeddings();

        // Verify that the embedding service was not called for this user
        const { EmbeddingService } = await import("@vrooli/server");
        const mockGetEmbeddings = EmbeddingService.get().getEmbeddings as vi.MockedFunction<typeof EmbeddingService.get>["getEmbeddings"];
        
        // The mock should not have been called with this user's data
        expect(mockGetEmbeddings).not.toHaveBeenCalledWith("User", expect.arrayContaining(["User with Current Embedding currentembedding Already has embedding"]));
    });

    it("should handle multiple translations for a single entity", async () => {
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Multilingual User",
                handle: "multilingual",
                isBot: false,
                translations: {
                    create: [
                        {
                            id: generatePK(),
                            language: "en",
                            bio: "English bio",
                            embeddingExpiredAt: null,
                        },
                        {
                            id: generatePK(),
                            language: "es",
                            bio: "Bio en español",
                            embeddingExpiredAt: null,
                        },
                        {
                            id: generatePK(),
                            language: "fr",
                            bio: "Bio en français",
                            embeddingExpiredAt: null,
                        },
                    ],
                },
            },
        });
        testUserIds.push(user.id);

        await generateEmbeddings();

        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: user.id },
            include: { translations: true },
        });

        // All translations should have embeddings
        expect(updatedUser?.translations).toHaveLength(3);
        for (const translation of updatedUser!.translations) {
            await validateEmbedding("user_translation", translation.id);
        }
    });

    // Skipping run status test - Run model doesn't have embedding fields in schema

    it("should skip deleted resource versions", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Deleted Resource Owner",
                handle: "deletedowner",
                isBot: false,
            },
        });
        testUserIds.push(owner.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id }
                },
                resourceType: "Routine",
                isDeleted: true, // Deleted resource
                isPrivate: false,
                isInternal: false,
            },
        });
        testResourceIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionLabel: "1.0.0",
                isDeleted: false,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Deleted Resource Version",
                        description: "Should not get embedding",
                        embeddingExpiredAt: null,
                    }],
                },
            },
            include: {
                translations: true,
            },
        });
        testResourceVersionIds.push(resourceVersion.id);

        await generateEmbeddings();

        // Should not have embedding because root resource is deleted
        await validateEmbedding("resource_translation", resourceVersion.translations[0].id, false);
    });
});