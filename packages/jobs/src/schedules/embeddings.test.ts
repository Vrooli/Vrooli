import { generatePK, generatePublicId, RunStatus } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { generateEmbeddings } from "./embeddings.js";

import { DbProvider } from "@vrooli/server/db/provider.js";

// Mock the EmbeddingService
vi.mock("@vrooli/server/services/embedding.js", () => ({
    EmbeddingService: {
        get: () => ({
            getEmbeddings: vi.fn().mockImplementation((objectType: string, sentences: string[]) => {
                // Return mock embeddings - arrays of 1536 dimensions (OpenAI standard)
                return sentences.map(() => Array(1536).fill(0.1));
            }),
        }),
        getEmbeddableString: vi.fn().mockImplementation((data: any, language?: string) => {
            // Simple mock implementation
            const parts = [];
            if (data.name) parts.push(data.name);
            if (data.handle) parts.push(data.handle);
            if (data.description) parts.push(data.description);
            if (data.bio) parts.push(data.bio);
            return parts.join(" ");
        }),
    },
}));

describe("generateEmbeddings integration tests", () => {
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
    const testRoutineIds: bigint[] = [];

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
        testRoutineIds.length = 0;
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
        if (testRoutineIds.length > 0) {
            await db.routine.deleteMany({ where: { id: { in: testRoutineIds } } });
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
        await generateEmbeddings();

        // Check that embeddings were updated
        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: user.id },
            include: { translations: true },
        });

        expect(updatedUser?.translations[0].embedding).toBeDefined();
        expect(updatedUser?.translations[0].embeddingExpiredAt).toBeDefined();
        expect(updatedUser?.translations[0].embeddingExpiredAt!.getTime()).toBeGreaterThan(Date.now() - 1000);
    });

    it("should generate embeddings for teams with bio translations", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam",
                name: "Test Team",
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
        });
        testTeamIds.push(team.id);

        await generateEmbeddings();

        const updatedTeam = await DbProvider.get().team.findUnique({
            where: { id: team.id },
            include: { translations: true },
        });

        expect(updatedTeam?.translations[0].embedding).toBeDefined();
    });

    it("should generate embeddings for chats with name and description", async () => {
        const creator = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Chat Creator",
                handle: "chatcreator",
            },
        });
        testUserIds.push(creator.id);

        const chat = await DbProvider.get().chat.create({
            data: {
                id: generatePK(),
                createdById: creator.id,
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
        });
        testChatIds.push(chat.id);

        await generateEmbeddings();

        const updatedChat = await DbProvider.get().chat.findUnique({
            where: { id: chat.id },
            include: { translations: true },
        });

        expect(updatedChat?.translations[0].embedding).toBeDefined();
    });

    it("should generate embeddings for issues", async () => {
        const creator = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Issue Creator",
                handle: "issuecreator",
            },
        });
        testUserIds.push(creator.id);

        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: creator.id,
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
        });
        testIssueIds.push(issue.id);

        await generateEmbeddings();

        const updatedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            include: { translations: true },
        });

        expect(updatedIssue?.translations[0].embedding).toBeDefined();
    });

    it("should generate embeddings for meetings", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Meeting Owner",
                handle: "meetingowner",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "meetingteam",
            },
        });
        testTeamIds.push(team.id);

        const meeting = await DbProvider.get().meeting.create({
            data: {
                id: generatePK(),
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
        });
        testMeetingIds.push(meeting.id);

        await generateEmbeddings();

        const updatedMeeting = await DbProvider.get().meeting.findUnique({
            where: { id: meeting.id },
            include: { translations: true },
        });

        expect(updatedMeeting?.translations[0].embedding).toBeDefined();
    });

    it("should generate embeddings for resource versions", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner",
            },
        });
        testUserIds.push(owner.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                resourceType: "RoutineVersion",
            },
        });
        testResourceIds.push(resource.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                resourceId: resource.id,
                versionLabel: "1.0.0",
                schemaLanguage: "json",
                schema: "{}",
                routineId: routine.id,
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
        });
        testResourceVersionIds.push(resourceVersion.id);

        await generateEmbeddings();

        const updatedVersion = await DbProvider.get().resource_version.findUnique({
            where: { id: resourceVersion.id },
            include: { translations: true },
        });

        expect(updatedVersion?.translations[0].embedding).toBeDefined();
    });

    it("should generate embeddings for tags", async () => {
        const creator = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Tag Creator",
                handle: "tagcreator",
            },
        });
        testUserIds.push(creator.id);

        const tag = await DbProvider.get().tag.create({
            data: {
                id: generatePK(),
                createdById: creator.id,
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
        });
        testTagIds.push(tag.id);

        await generateEmbeddings();

        const updatedTag = await DbProvider.get().tag.findUnique({
            where: { id: tag.id },
            include: { translations: true },
        });

        expect(updatedTag?.translations[0].embedding).toBeDefined();
    });

    it("should generate embeddings for reminders (non-translatable)", async () => {
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reminder User",
                handle: "reminderuser",
            },
        });
        testUserIds.push(user.id);

        const reminder = await DbProvider.get().reminder.create({
            data: {
                id: generatePK(),
                userId: user.id,
                name: "Test Reminder",
                description: "Remember to test embeddings",
                dueDate: new Date(Date.now() + 86400000), // Tomorrow
                index: 0,
                embeddingExpiredAt: null,
            },
        });
        testReminderIds.push(reminder.id);

        await generateEmbeddings();

        const updatedReminder = await DbProvider.get().reminder.findUnique({
            where: { id: reminder.id },
        });

        expect(updatedReminder?.embedding).toBeDefined();
        expect(updatedReminder?.embeddingExpiredAt).toBeDefined();
    });

    it("should generate embeddings for runs (non-translatable)", async () => {
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Run User",
                handle: "runuser",
            },
        });
        testUserIds.push(user.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: user.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                userId: user.id,
                routineId: routine.id,
                status: RunStatus.InProgress,
                name: "Test Run in Progress",
                embeddingExpiredAt: null,
            },
        });
        testRunIds.push(run.id);

        await generateEmbeddings();

        const updatedRun = await DbProvider.get().run.findUnique({
            where: { id: run.id },
        });

        expect(updatedRun?.embedding).toBeDefined();
    });

    it("should skip entities with current embeddings", async () => {
        const futureDate = new Date(Date.now() + 86400000); // Tomorrow
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "User with Current Embedding",
                handle: "currentembedding",
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        bio: "Already has embedding",
                        embedding: [0.1, 0.2, 0.3], // Mock embedding
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
        const { EmbeddingService } = await import("../../../server/src/services/embedding.ts");
        const mockGetEmbeddings = EmbeddingService.get().getEmbeddings as any;
        
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
        updatedUser?.translations.forEach(translation => {
            expect(translation.embedding).toBeDefined();
            expect(translation.embeddingExpiredAt).toBeDefined();
        });
    });

    it("should only process runs with specific statuses", async () => {
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Run Status User",
                handle: "runstatususer",
            },
        });
        testUserIds.push(user.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: user.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        // Create runs with different statuses
        const scheduledRun = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                userId: user.id,
                routineId: routine.id,
                status: RunStatus.Scheduled,
                name: "Scheduled Run",
                embeddingExpiredAt: null,
            },
        });
        testRunIds.push(scheduledRun.id);

        const completedRun = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                userId: user.id,
                routineId: routine.id,
                status: RunStatus.Completed,
                name: "Completed Run",
                embeddingExpiredAt: null,
            },
        });
        testRunIds.push(completedRun.id);

        await generateEmbeddings();

        const updatedScheduled = await DbProvider.get().run.findUnique({
            where: { id: scheduledRun.id },
        });
        const updatedCompleted = await DbProvider.get().run.findUnique({
            where: { id: completedRun.id },
        });

        // Only scheduled/in-progress runs should get embeddings
        expect(updatedScheduled?.embedding).toBeDefined();
        expect(updatedCompleted?.embedding).toBeNull();
    });

    it("should skip deleted resource versions", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Deleted Resource Owner",
                handle: "deletedowner",
            },
        });
        testUserIds.push(owner.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                resourceType: "RoutineVersion",
                isDeleted: true, // Deleted resource
            },
        });
        testResourceIds.push(resource.id);

        const routine = await DbProvider.get().routine.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                isPrivate: false,
                isInternal: false,
            },
        });
        testRoutineIds.push(routine.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                resourceId: resource.id,
                versionLabel: "1.0.0",
                schemaLanguage: "json",
                schema: "{}",
                routineId: routine.id,
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
        });
        testResourceVersionIds.push(resourceVersion.id);

        await generateEmbeddings();

        const updatedVersion = await DbProvider.get().resource_version.findUnique({
            where: { id: resourceVersion.id },
            include: { translations: true },
        });

        // Should not have embedding because root resource is deleted
        expect(updatedVersion?.translations[0].embedding).toBeNull();
    });
});