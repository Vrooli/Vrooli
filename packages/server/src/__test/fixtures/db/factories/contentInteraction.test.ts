import { describe, it, expect } from "vitest";
import { CommentFor, BookmarkFor, ViewFor, generatePK } from "@vrooli/shared";
import {
    CommentDbFactory,
    TagDbFactory,
    BookmarkDbFactory,
    BookmarkListDbFactory,
    ViewDbFactory,
} from "./contentInteraction.js";

// Generate realistic snowflake IDs for testing
const testIds = {
    user1: generatePK(),
    user2: generatePK(),
    user3: generatePK(),
    issue1: generatePK(),
    pr1: generatePK(),
    rv1: generatePK(),
    project1: generatePK(),
    resource1: generatePK(),
    team1: generatePK(),
    session1: generatePK(),
    list1: generatePK(),
};

describe("Content & Interaction Factory Tests", () => {
    describe("CommentDbFactory", () => {
        it("should create minimal comment", () => {
            const comment = CommentDbFactory.createMinimal();
            expect(comment.id).toBeDefined();
            expect(comment.score).toBe(0);
            expect(comment.bookmarks).toBe(0);
            expect(comment.translations).toBeDefined();
        });

        it("should create comment for specific object types", () => {
            const issueComment = CommentDbFactory.createForObject(
                CommentFor.Issue,
                testIds.issue1.toString()
            );
            expect(issueComment.issueId).toBeDefined();
            expect(issueComment.issueId.toString()).toBe(testIds.issue1.toString());

            const prComment = CommentDbFactory.createForObject(
                CommentFor.PullRequest,
                testIds.pr1.toString()
            );
            expect(prComment.pullRequestId).toBeDefined();
            expect(prComment.pullRequestId.toString()).toBe(testIds.pr1.toString());
        });

        it("should create threaded comments", () => {
            const parentComment = CommentDbFactory.createThreaded(testIds.user1.toString());
            const reply = CommentDbFactory.createReply(parentComment.id.toString());
            
            expect(reply.parentId).toBeDefined();
            expect(reply.parentId.toString()).toBe(parentComment.id.toString());
        });

        it("should create bulk comments for various objects", () => {
            const comments = CommentDbFactory.createBulkForObjects([
                { type: CommentFor.Issue, id: testIds.issue1.toString(), userId: testIds.user1.toString() },
                { type: CommentFor.PullRequest, id: testIds.pr1.toString(), userId: testIds.user2.toString() },
                { type: CommentFor.ResourceVersion, id: testIds.rv1.toString(), userId: testIds.user3.toString() },
            ]);

            expect(comments).toHaveLength(3);
            expect(comments[0].issueId).toBeDefined();
            expect(comments[1].pullRequestId).toBeDefined();
            expect(comments[2].resourceVersionId).toBeDefined();
        });

        it("should validate comment fixtures", () => {
            const factory = new CommentDbFactory();
            const validation = factory.validateFixture(factory.createMinimal());
            
            expect(validation.isValid).toBe(true);
            expect(validation.errors).toHaveLength(0);
        });

        it("should create invalid comment scenarios", () => {
            const factory = new CommentDbFactory();
            const invalid = factory.createInvalid("missingRequired");
            
            expect(invalid.translations).toBeUndefined();
        });
    });

    describe("TagDbFactory", () => {
        it("should create minimal tag", () => {
            const tag = TagDbFactory.createMinimal();
            expect(tag.id).toBeDefined();
            expect(tag.publicId).toBeDefined();
            expect(tag.tag).toBeDefined();
            expect(tag.bookmarks).toBe(0);
        });

        it("should create popular tags", () => {
            const popularTag = TagDbFactory.createPopular({
                tag: "trending",
                bookmarks: 500,
            });
            expect(popularTag.bookmarks).toBe(500);
        });

        it("should create hierarchical tags", () => {
            const tags = TagDbFactory.createHierarchical(
                "programming",
                ["javascript", "typescript", "python"]
            );
            
            expect(tags).toHaveLength(4); // 1 parent + 3 children
            expect(tags[0].tag).toBe("programming");
            expect(tags[1].parentId).toBeDefined();
            expect(tags[1].parentId.toString()).toBe(tags[0].id.toString());
        });

        it("should create tag cloud with varying popularity", () => {
            const tagCloud = TagDbFactory.createTagCloud([
                { name: "react", popularity: 150 },
                { name: "vue", popularity: 100 },
                { name: "angular", popularity: 80 },
            ]);

            expect(tagCloud).toHaveLength(3);
            expect(tagCloud[0].bookmarks).toBe(150);
            expect(tagCloud[1].bookmarks).toBe(100);
            expect(tagCloud[2].bookmarks).toBe(80);
        });

        it("should create multilingual tags", () => {
            const multilingualTag = TagDbFactory.createMultilingualTag(
                "technology",
                {
                    en: "Technology and innovation",
                    es: "Tecnología e innovación",
                    fr: "Technologie et innovation",
                }
            );

            expect(multilingualTag.tag).toBe("technology");
            expect(multilingualTag.translations.create).toHaveLength(3);
        });
    });

    describe("BookmarkDbFactory", () => {
        it("should create minimal bookmark", () => {
            const bookmark = BookmarkDbFactory.createMinimal(testIds.user1.toString());
            expect(bookmark.id).toBeDefined();
            expect(bookmark.publicId).toBeDefined();
            expect(bookmark.by.connect.id).toBe(testIds.user1.toString());
        });

        it("should create bookmarks for different object types", () => {
            // Using BookmarkFor enum
            const commentBookmark = BookmarkDbFactory.createForObject(
                testIds.user1.toString(),
                testIds.issue1.toString(),
                BookmarkFor.Comment
            );
            expect(commentBookmark.comment.connect.id).toBe(testIds.issue1.toString());

            // Using additional supported types
            const projectBookmark = BookmarkDbFactory.createForObject(
                testIds.user1.toString(),
                testIds.project1.toString(),
                "Project"
            );
            expect(projectBookmark.project.connect.id).toBe(testIds.project1.toString());
        });

        it("should create bookmark in a list", () => {
            const bookmark = BookmarkDbFactory.createInList(
                testIds.user1.toString(),
                testIds.list1.toString(),
                testIds.rv1.toString(),
                "Routine"
            );
            
            expect(bookmark.list.connect.id).toBe(testIds.list1.toString());
            expect(bookmark.routine.connect.id).toBe(testIds.rv1.toString());
        });

        it("should validate bookmark with multiple objects (invalid)", () => {
            const factory = new BookmarkDbFactory();
            const multipleObjects = factory.createInvalid("multipleObjects");
            const validation = factory.validateFixture(multipleObjects);
            
            expect(validation.isValid).toBe(false);
            expect(validation.errors).toContain("Bookmark cannot reference multiple objects");
        });
    });

    describe("BookmarkListDbFactory", () => {
        it("should create minimal bookmark list", () => {
            const list = BookmarkListDbFactory.createMinimal(testIds.user1.toString());
            expect(list.id).toBeDefined();
            expect(list.publicId).toBeDefined();
            expect(list.isPrivate).toBe(false);
            expect(list.createdBy.connect.id).toBe(testIds.user1.toString());
        });

        it("should create private bookmark list", () => {
            const privateList = BookmarkListDbFactory.createPrivateList(testIds.user1.toString());
            expect(privateList.isPrivate).toBe(true);
        });

        it("should create list with bookmarks of different types", () => {
            const listWithBookmarks = BookmarkListDbFactory.createWithBookmarks(
                testIds.user1.toString(),
                [
                    { objectId: testIds.project1.toString(), objectType: "Project" },
                    { objectId: testIds.user2.toString(), objectType: BookmarkFor.User },
                    { objectId: testIds.issue1.toString(), objectType: BookmarkFor.Tag },
                ]
            );

            expect(listWithBookmarks.bookmarks.create).toHaveLength(3);
        });

        it("should create shared list", () => {
            const sharedList = BookmarkListDbFactory.createSharedList(
                testIds.user1.toString(),
                "Shared Resources",
                [testIds.user2.toString(), testIds.user3.toString()]
            );

            expect(sharedList.isPrivate).toBe(false);
            expect(sharedList.translations.create[0].name).toBe("Shared Resources");
        });
    });

    describe("ViewDbFactory", () => {
        it("should create minimal view", () => {
            const view = ViewDbFactory.createMinimal(testIds.user1.toString());
            expect(view.id).toBeDefined();
            expect(view.name).toBe("Test View");
            expect(view.lastViewedAt).toBeDefined();
            expect(view.byId.toString()).toBe(testIds.user1.toString());
        });

        it("should create views for different object types", () => {
            // Using ViewFor enum
            const resourceView = ViewDbFactory.createForObject(
                testIds.user1.toString(),
                testIds.resource1.toString(),
                ViewFor.Resource
            );
            expect(resourceView.resourceId.toString()).toBe(testIds.resource1.toString());

            // Using legacy type
            const issueView = ViewDbFactory.createForObject(
                testIds.user1.toString(),
                testIds.issue1.toString(),
                "Issue"
            );
            expect(issueView.issueId.toString()).toBe(testIds.issue1.toString());
        });

        it("should create anonymous views", () => {
            const anonView = ViewDbFactory.createAnonymousView(
                testIds.session1.toString(),
                testIds.team1.toString(),
                ViewFor.Team
            );

            expect(anonView.bySessionId.toString()).toBe(testIds.session1.toString());
            expect(anonView.byId).toBeUndefined();
            expect(anonView.teamId.toString()).toBe(testIds.team1.toString());
        });

        it("should create view history", () => {
            const history = ViewDbFactory.createViewHistory(testIds.user1.toString(), [
                { id: testIds.resource1.toString(), type: ViewFor.Resource },
                { id: testIds.team1.toString(), type: ViewFor.Team },
                { id: testIds.user2.toString(), type: ViewFor.User },
            ]);

            expect(history).toHaveLength(3);
            expect(history[0].lastViewedAt).toBeInstanceOf(Date);
            expect(history[1].lastViewedAt.getTime()).toBeLessThan(history[0].lastViewedAt.getTime());
        });

        it("should validate view without viewer (invalid)", () => {
            const factory = new ViewDbFactory();
            const noViewer = factory.createInvalid("missingRequired");
            const validation = factory.validateFixture(noViewer);
            
            expect(validation.isValid).toBe(false);
            expect(validation.errors).toContain("View must have either byId (user) or bySessionId");
        });

        it("should create views with specific timestamps", () => {
            const pastDate = new Date("2023-01-01");
            const view = ViewDbFactory.createWithTimestamp(testIds.user1.toString(), pastDate);
            
            expect(view.lastViewedAt).toEqual(pastDate);
        });
    });

    describe("Cross-Factory Integration", () => {
        it("should create a complete interaction scenario", () => {
            // User views a resource
            const view = ViewDbFactory.createForObject(
                testIds.user1.toString(),
                testIds.resource1.toString(),
                ViewFor.Resource
            );

            // User bookmarks the resource
            const bookmark = BookmarkDbFactory.createForObject(
                testIds.user1.toString(),
                testIds.resource1.toString(),
                BookmarkFor.Resource
            );

            // User comments on the resource (if it were supported)
            // Note: Comments don't support Resource directly, so we use ResourceVersion
            const comment = CommentDbFactory.createForObject(
                CommentFor.ResourceVersion,
                testIds.rv1.toString(),
                {
                    ownedByUserId: testIds.user1,
                }
            );

            // Tag the resource
            const tags = TagDbFactory.createTagCloud([
                { name: "useful", popularity: 50 },
                { name: "tutorial", popularity: 30 },
            ]);

            expect(view).toBeDefined();
            expect(bookmark).toBeDefined();
            expect(comment).toBeDefined();
            expect(tags).toHaveLength(2);
        });

        it("should create bookmarks for tagged content", () => {
            // Create some tags
            const tags = TagDbFactory.createHierarchical(
                "programming",
                ["javascript", "typescript"]
            );

            // Create bookmarks for the tags
            const tagBookmarks = tags.map(tag =>
                BookmarkDbFactory.createForObject(
                    testIds.user1.toString(),
                    tag.id.toString(),
                    BookmarkFor.Tag
                )
            );

            expect(tagBookmarks).toHaveLength(3);
            expect(tagBookmarks[0].tag.connect.id).toBe(tags[0].id.toString());
        });
    });
});