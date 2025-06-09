import { generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { countReacts } from "./countReacts.js";

// Direct import to avoid problematic services
const { DbProvider } = await import("../../../server/src/db/provider.ts");

describe("countReacts integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testChatIds: bigint[] = [];
    const testChatMessageIds: bigint[] = [];
    const testCommentIds: bigint[] = [];
    const testIssueIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testReactionIds: bigint[] = [];
    const testReactionSummaryIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testChatIds.length = 0;
        testChatMessageIds.length = 0;
        testCommentIds.length = 0;
        testIssueIds.length = 0;
        testResourceIds.length = 0;
        testReactionIds.length = 0;
        testReactionSummaryIds.length = 0;
    });

    afterEach(async () => {
        // Clean up test data using collected IDs
        await DbProvider.get().$transaction([
            DbProvider.get().reaction.deleteMany({
                where: { id: { in: testReactionIds } },
            }),
            DbProvider.get().reactionSummary.deleteMany({
                where: { id: { in: testReactionSummaryIds } },
            }),
            DbProvider.get().chat_message.deleteMany({
                where: { id: { in: testChatMessageIds } },
            }),
            DbProvider.get().comment.deleteMany({
                where: { id: { in: testCommentIds } },
            }),
            DbProvider.get().issue.deleteMany({
                where: { id: { in: testIssueIds } },
            }),
            DbProvider.get().resource.deleteMany({
                where: { id: { in: testResourceIds } },
            }),
            DbProvider.get().chat.deleteMany({
                where: { id: { in: testChatIds } },
            }),
            DbProvider.get().user.deleteMany({
                where: { id: { in: testUserIds } },
            }),
        ]);
    });

    it("should update reaction scores and summaries when they mismatch", async () => {
        // Create test users
        const reactor1 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reactor 1",
                handle: "reactor1",
            },
        });
        testUserIds.push(reactor1.id);

        const reactor2 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reactor 2",
                handle: "reactor2",
            },
        });
        testUserIds.push(reactor2.id);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: "owner1",
            },
        });
        testUserIds.push(owner.id);

        // Create test issue with incorrect score
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                score: 100, // Incorrect score
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Issue",
                        description: "Test Description",
                    }],
                },
            },
        });
        testIssueIds.push(issue.id);

        // Create actual reactions (👍 = +1, ❤️ = +2)
        const reaction1Id = generatePK();
        const reaction2Id = generatePK();
        testReactionIds.push(reaction1Id, reaction2Id);
        await DbProvider.get().reaction.createMany({
            data: [
                {
                    id: reaction1Id,
                    byId: reactor1.id,
                    issueId: issue.id,
                    emoji: "👍",
                },
                {
                    id: reaction2Id,
                    byId: reactor2.id,
                    issueId: issue.id,
                    emoji: "❤️",
                },
            ],
        });

        // Run the count reacts function
        await countReacts();

        // Check that the score was updated correctly (1 + 2 = 3)
        const updatedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            include: { reactionSummaries: true },
        });

        expect(updatedIssue?.score).toBe(3);
        expect(updatedIssue?.reactionSummaries).toHaveLength(2);
        
        const summaryMap = new Map(updatedIssue?.reactionSummaries.map(s => [s.emoji, s.count]) || []);
        expect(summaryMap.get("👍")).toBe(1);
        expect(summaryMap.get("❤️")).toBe(1);

        // Track summary IDs for cleanup
        updatedIssue?.reactionSummaries.forEach(s => testReactionSummaryIds.push(s.id));
    });

    it("should create reaction summaries when none exist", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 2",
                handle: "owner2",
            },
        });
        testUserIds.push(owner.id);

        // Create test resource with no reaction summaries
        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                resourceType: "RoutineVersion",
                score: 0,
            },
        });
        testResourceIds.push(resource.id);

        // Create reactions
        const reaction1Id = generatePK();
        const reaction2Id = generatePK();
        const reaction3Id = generatePK();
        testReactionIds.push(reaction1Id, reaction2Id, reaction3Id);
        await DbProvider.get().reaction.createMany({
            data: [
                {
                    id: reaction1Id,
                    byId: owner.id,
                    resourceId: resource.id,
                    emoji: "👍",
                },
                {
                    id: reaction2Id,
                    byId: owner.id,
                    resourceId: resource.id,
                    emoji: "👍",
                },
                {
                    id: reaction3Id,
                    byId: owner.id,
                    resourceId: resource.id,
                    emoji: "🎉",
                },
            ],
        });

        // Run the count reacts function
        await countReacts();

        // Check that reaction summaries were created
        const updatedResource = await DbProvider.get().resource.findUnique({
            where: { id: resource.id },
            include: { reactionSummaries: true },
        });

        expect(updatedResource?.score).toBe(5); // 2 * 1 + 1 * 3 = 5
        expect(updatedResource?.reactionSummaries).toHaveLength(2);
        
        const summaryMap = new Map(updatedResource?.reactionSummaries.map(s => [s.emoji, s.count]) || []);
        expect(summaryMap.get("👍")).toBe(2);
        expect(summaryMap.get("🎉")).toBe(1);

        // Track summary IDs for cleanup
        updatedResource?.reactionSummaries.forEach(s => testReactionSummaryIds.push(s.id));
    });

    it("should update existing reaction summaries", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 3",
                handle: "owner3",
            },
        });
        testUserIds.push(owner.id);

        // Create test comment with existing reaction summaries
        const summary1Id = generatePK();
        const summary2Id = generatePK();
        testReactionSummaryIds.push(summary1Id, summary2Id);
        
        const comment = await DbProvider.get().comment.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                issueId: generatePK(), // Valid bigint foreign key
                body: "Test comment",
                score: 10,
                reactionSummaries: {
                    create: [
                        {
                            id: summary1Id,
                            emoji: "👍",
                            count: 5, // Incorrect count
                        },
                        {
                            id: summary2Id,
                            emoji: "❤️",
                            count: 3, // Will be removed
                        },
                    ],
                },
            },
        });
        testCommentIds.push(comment.id);

        // Create actual reactions (only thumbs up)
        const reaction1Id = generatePK();
        const reaction2Id = generatePK();
        testReactionIds.push(reaction1Id, reaction2Id);
        await DbProvider.get().reaction.createMany({
            data: [
                {
                    id: reaction1Id,
                    byId: owner.id,
                    commentId: comment.id,
                    emoji: "👍",
                },
                {
                    id: reaction2Id,
                    byId: owner.id,
                    commentId: comment.id,
                    emoji: "👍",
                },
            ],
        });

        // Run the count reacts function
        await countReacts();

        // Check that reaction summaries were updated
        const updatedComment = await DbProvider.get().comment.findUnique({
            where: { id: comment.id },
            include: { reactionSummaries: true },
        });

        expect(updatedComment?.score).toBe(2); // 2 * 1 = 2
        expect(updatedComment?.reactionSummaries).toHaveLength(1);
        expect(updatedComment?.reactionSummaries[0].emoji).toBe("👍");
        expect(updatedComment?.reactionSummaries[0].count).toBe(2);
    });

    it("should handle duplicate reaction summaries", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 4",
                handle: "owner4",
            },
        });
        testUserIds.push(owner.id);

        const chat = await DbProvider.get().chat.create({
            data: {
                id: generatePK(),
                participantsCreate: {
                    create: [
                        {
                            id: generatePK(),
                            userId: owner.id,
                        },
                    ],
                },
                createdById: owner.id,
            },
        });
        testChatIds.push(chat.id);

        // Create chat message with duplicate reaction summaries
        const summary1Id = generatePK();
        const summary2Id = generatePK();
        testReactionSummaryIds.push(summary1Id, summary2Id);
        
        const chatMessage = await DbProvider.get().chat_message.create({
            data: {
                id: generatePK(),
                chatId: chat.id,
                userId: owner.id,
                body: "Test message",
                score: 10,
                reactionSummaries: {
                    create: [
                        {
                            id: summary1Id,
                            emoji: "👍",
                            count: 2,
                        },
                        {
                            id: summary2Id,
                            emoji: "👍", // Duplicate emoji
                            count: 3,
                        },
                    ],
                },
            },
        });
        testChatMessageIds.push(chatMessage.id);

        // Create actual reactions
        const reactionId = generatePK();
        testReactionIds.push(reactionId);
        await DbProvider.get().reaction.create({
            data: {
                id: reactionId,
                byId: owner.id,
                chatMessageId: chatMessage.id,
                emoji: "👍",
            },
        });

        // Run the count reacts function
        await countReacts();

        // Check that duplicates were removed
        const updatedMessage = await DbProvider.get().chat_message.findUnique({
            where: { id: chatMessage.id },
            include: { reactionSummaries: true },
        });

        expect(updatedMessage?.score).toBe(1);
        expect(updatedMessage?.reactionSummaries).toHaveLength(1);
        expect(updatedMessage?.reactionSummaries[0].emoji).toBe("👍");
        expect(updatedMessage?.reactionSummaries[0].count).toBe(1);
    });

    it("should handle entities with no reactions", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 5",
                handle: "owner5",
            },
        });
        testUserIds.push(owner.id);

        // Create entities with scores but no actual reactions
        const summaryId = generatePK();
        testReactionSummaryIds.push(summaryId);
        
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                score: 15,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Issue with no reactions",
                        description: "Should be reset to 0",
                    }],
                },
                reactionSummaries: {
                    create: [
                        {
                            id: summaryId,
                            emoji: "🚀",
                            count: 5,
                        },
                    ],
                },
            },
        });
        testIssueIds.push(issue.id);

        // Run the count reacts function
        await countReacts();

        // Check that the score and summaries were cleared
        const correctedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
            include: { reactionSummaries: true },
        });

        expect(correctedIssue?.score).toBe(0);
        expect(correctedIssue?.reactionSummaries).toHaveLength(0);
    });

    it("should process all supported entity types", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner 6",
                handle: "owner6",
            },
        });
        testUserIds.push(owner.id);

        const chat = await DbProvider.get().chat.create({
            data: {
                id: generatePK(),
                participantsCreate: {
                    create: [
                        {
                            id: generatePK(),
                            userId: owner.id,
                        },
                    ],
                },
                createdById: owner.id,
            },
        });
        testChatIds.push(chat.id);

        // Create one entity of each type with incorrect scores
        const chatMessage = await DbProvider.get().chat_message.create({
            data: {
                id: generatePK(),
                chatId: chat.id,
                userId: owner.id,
                body: "Test",
                score: 5,
            },
        });
        testChatMessageIds.push(chatMessage.id);

        const comment = await DbProvider.get().comment.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                issueId: generatePK(),
                body: "Test",
                score: 5,
            },
        });
        testCommentIds.push(comment.id);

        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                score: 5,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test",
                        description: "Test",
                    }],
                },
            },
        });
        testIssueIds.push(issue.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                resourceType: "RoutineVersion",
                score: 5,
            },
        });
        testResourceIds.push(resource.id);

        // Run the count reacts function
        await countReacts();

        // Check that all scores were corrected to 0
        const results = await DbProvider.get().$transaction([
            DbProvider.get().chat_message.findUnique({ where: { id: chatMessage.id }, select: { score: true } }),
            DbProvider.get().comment.findUnique({ where: { id: comment.id }, select: { score: true } }),
            DbProvider.get().issue.findUnique({ where: { id: issue.id }, select: { score: true } }),
            DbProvider.get().resource.findUnique({ where: { id: resource.id }, select: { score: true } }),
        ]);

        results.forEach(result => {
            expect(result?.score).toBe(0);
        });
    });
});
