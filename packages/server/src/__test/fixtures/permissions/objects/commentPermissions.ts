/**
 * Comment Permission Fixtures
 * 
 * Comprehensive permission testing fixtures for Comment objects.
 * Comments can be attached to Issues, PullRequests, and ResourceVersions.
 */

import { generatePK, DUMMY_ID, type CommentFor } from "@vrooli/shared";
import { ObjectPermissionFactory } from "../factories/ObjectPermissionFactory.js";

/**
 * Constants for comment generation
 */
const USER_ID_SUFFIX_LENGTH = 16;

// Minimal comment object for testing
function createMinimalComment(overrides: any = {}) {
    return {
        __typename: "Comment",
        id: generatePK(),
        owner: {
            __typename: "User",
            id: "222222222222222222",
        },
        commentedOn: {
            __typename: "Issue",
            id: DUMMY_ID,
        },
        translations: [{
            __typename: "CommentTranslation",
            id: generatePK(),
            language: "en",
            text: "Test comment",
        }],
        ...overrides,
    };
}

// Complete comment object for testing
function createCompleteComment(overrides: any = {}) {
    return {
        __typename: "Comment",
        id: generatePK(),
        owner: {
            __typename: "User",
            id: "222222222222222222",
            handle: "commenter",
            name: "Test Commenter",
        },
        commentedOn: {
            __typename: "Issue",
            id: DUMMY_ID,
            title: "Test Issue",
            isPublic: true,
            owner: { id: "333333333333333333" },
        },
        translations: [{
            __typename: "CommentTranslation",
            id: generatePK(),
            language: "en",
            text: "This is a comprehensive test comment with detailed content.",
        }],
        score: 5,
        bookmarks: 2,
        reportsCount: 0,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        ...overrides,
    };
}

// Create comment permission factory
export const commentPermissionFactory = new ObjectPermissionFactory({
    objectType: "Comment",
    createMinimal: createMinimalComment,
    createComplete: createCompleteComment,
    supportedActions: ["read", "create", "update", "delete", "report", "vote"],
    canBeTeamOwned: false, // Comments are always user-owned
    hasVisibility: false, // Comments inherit visibility from their parent
    customRules: {
        // Custom rule: can only create comments on objects you can read
        create: (session, comment) => {
            const parentObject = comment.commentedOn;
            
            // If parent is public, anyone can comment
            if (parentObject.isPublic) {
                return true;
            }
            
            // If user owns the parent object, they can comment
            if ("id" in session && parentObject.owner?.id === session.id) {
                return true;
            }
            
            // If it's a team object and user is a team member
            if (parentObject.team && "id" in session && session._testTeamMembership) {
                const membership = session._testTeamMembership as any;
                if (membership.teamId === parentObject.team.id) {
                    return true;
                }
            }
            
            return false;
        },
        
        // Custom rule: can read comments if you can read the parent object
        read: (session, comment) => {
            const parentObject = comment.commentedOn;
            
            // Public parent objects = public comments
            if (parentObject.isPublic) {
                return true;
            }
            
            // Owner can always read their own comments
            if ("id" in session && comment.owner?.id === session.id) {
                return true;
            }
            
            // If you can read the parent, you can read comments on it
            if ("id" in session && parentObject.owner?.id === session.id) {
                return true;
            }
            
            // Team member access
            if (parentObject.team && "id" in session && session._testTeamMembership) {
                const membership = session._testTeamMembership as any;
                if (membership.teamId === parentObject.team.id) {
                    return true;
                }
            }
            
            // Admins can read all comments
            if ("roles" in session && session.roles?.some(r => r.role.name === "Admin")) {
                return true;
            }
            
            return false;
        },
        
        // Custom rule: can only update your own comments
        update: (session, comment) => {
            if ("id" in session && comment.owner?.id === session.id) {
                return true;
            }
            
            // Admins can update any comment
            if ("roles" in session && session.roles?.some(r => r.role.name === "Admin")) {
                return true;
            }
            
            return false;
        },
        
        // Custom rule: can delete your own comments, or admin can delete any
        delete: (session, comment) => {
            if ("id" in session && comment.owner?.id === session.id) {
                return true;
            }
            
            // Admins can delete any comment
            if ("roles" in session && session.roles?.some(r => r.role.name === "Admin")) {
                return true;
            }
            
            // Parent object owner can delete comments on their objects
            const parentObject = comment.commentedOn;
            if ("id" in session && parentObject.owner?.id === session.id) {
                return true;
            }
            
            return false;
        },
        
        // Custom rule: anyone can report comments they can read
        report: (session, comment) => {
            // Must be able to read the comment to report it
            return commentPermissionFactory.config.customRules!.read!(session, comment);
        },
        
        // Custom rule: anyone can vote on comments they can read
        vote: (session, comment) => {
            // Must be able to read the comment to vote on it
            // Cannot vote on your own comments
            if ("id" in session && comment.owner?.id === session.id) {
                return false;
            }
            
            return commentPermissionFactory.config.customRules!.read!(session, comment);
        },
    },
});

// Pre-configured comment scenarios
export const commentScenarios = {
    /**
     * Comment on a public issue
     */
    publicIssueComment: {
        id: "comment_public_issue",
        description: "Comment on a public issue",
        resource: createCompleteComment({
            commentedOn: {
                __typename: "Issue",
                id: DUMMY_ID,
                title: "Public Issue",
                isPublic: true,
                owner: { id: "333333333333333333" },
            },
        }),
        actors: [
            {
                id: "comment_owner",
                session: { id: "222222222222222222" } as any,
                permissions: {
                    read: true,
                    create: true,
                    update: true,
                    delete: true,
                    report: false, // Can't report own comment
                    vote: false, // Can't vote on own comment
                },
            },
            {
                id: "issue_owner",
                session: { id: "333333333333333333" } as any,
                permissions: {
                    read: true,
                    create: true,
                    update: false, // Can't update other's comment
                    delete: true, // Can delete comments on own issue
                    report: true,
                    vote: true,
                },
            },
            {
                id: "other_user",
                session: { id: "444444444444444444" } as any,
                permissions: {
                    read: true,
                    create: true,
                    update: false,
                    delete: false,
                    report: true,
                    vote: true,
                },
            },
        ],
        actions: ["read", "create", "update", "delete", "report", "vote"],
    },
    
    /**
     * Comment on a private resource version
     */
    privateResourceComment: {
        id: "comment_private_resource",
        description: "Comment on a private resource version",
        resource: createCompleteComment({
            commentedOn: {
                __typename: "ResourceVersion",
                id: DUMMY_ID,
                name: "Private Resource",
                isPublic: false,
                owner: { id: "333333333333333333" },
            },
        }),
        actors: [
            {
                id: "comment_owner",
                session: { id: "222222222222222222" } as any,
                permissions: {
                    read: false, // Can't read comment on private resource they can't access
                    create: false,
                    update: false,
                    delete: false,
                    report: false,
                    vote: false,
                },
            },
            {
                id: "resource_owner",
                session: { id: "333333333333333333" } as any,
                permissions: {
                    read: true,
                    create: true,
                    update: false,
                    delete: true,
                    report: true,
                    vote: true,
                },
            },
        ],
        actions: ["read", "create", "update", "delete", "report", "vote"],
    },
    
    /**
     * Team pull request comment
     */
    teamPullRequestComment: {
        id: "comment_team_pull_request",
        description: "Comment on team pull request",
        resource: createCompleteComment({
            commentedOn: {
                __typename: "PullRequest",
                id: DUMMY_ID,
                title: "Team PR",
                isPublic: false,
                team: { id: "team_123" },
            },
        }),
        actors: [
            {
                id: "team_member",
                session: { 
                    id: "222222222222222222",
                    _testTeamMembership: { teamId: "team_123", role: "Member" },
                } as any,
                permissions: {
                    read: true,
                    create: true,
                    update: true, // Own comment
                    delete: true, // Own comment
                    report: false,
                    vote: false,
                },
            },
            {
                id: "non_member",
                session: { id: "444444444444444444" } as any,
                permissions: {
                    read: false,
                    create: false,
                    update: false,
                    delete: false,
                    report: false,
                    vote: false,
                },
            },
        ],
        actions: ["read", "create", "update", "delete", "report", "vote"],
    },
    
    /**
     * Comment thread (nested comments)
     */
    commentThread: {
        id: "comment_thread",
        description: "Nested comment thread",
        resource: createCompleteComment({
            parent: {
                __typename: "Comment",
                id: "parent_comment_id",
                owner: { id: "333333333333333333" },
            },
        }),
        actors: [
            {
                id: "thread_starter",
                session: { id: "333333333333333333" } as any,
                permissions: {
                    read: true,
                    create: true,
                    update: false, // Different comment
                    delete: false, // Different comment
                    report: true,
                    vote: true,
                },
            },
            {
                id: "reply_author",
                session: { id: "222222222222222222" } as any,
                permissions: {
                    read: true,
                    create: true,
                    update: true, // Own reply
                    delete: true, // Own reply
                    report: false,
                    vote: false,
                },
            },
        ],
        actions: ["read", "create", "update", "delete", "report", "vote"],
    },
};

// Helper functions for testing comment permissions
export const commentPermissionHelpers = {
    /**
     * Create a comment on a specific object type
     */
    createCommentOn: (
        objectType: CommentFor,
        objectId: string,
        commentAuthorId: string,
        objectIsPublic = true,
        objectOwnerId?: string,
    ) => {
        return createCompleteComment({
            owner: { id: commentAuthorId },
            commentedOn: {
                __typename: objectType,
                id: objectId,
                isPublic: objectIsPublic,
                owner: objectOwnerId ? { id: objectOwnerId } : undefined,
            },
        });
    },
    
    /**
     * Test if user can comment on an object
     */
    canUserCommentOn: (userId: string, targetObject: any) => {
        // Can comment on public objects
        if (targetObject.isPublic) {
            return true;
        }
        
        // Can comment on own objects
        if (targetObject.owner?.id === userId) {
            return true;
        }
        
        // Can comment on team objects if team member
        if (targetObject.team && targetObject.team.members?.some((m: any) => m.userId === userId)) {
            return true;
        }
        
        return false;
    },
    
    /**
     * Test if user can read a comment
     */
    canUserReadComment: (userId: string, comment: any) => {
        // Can read own comments
        if (comment.owner?.id === userId) {
            return true;
        }
        
        // Can read comments on public objects
        if (comment.commentedOn?.isPublic) {
            return true;
        }
        
        // Can read comments on own objects
        if (comment.commentedOn?.owner?.id === userId) {
            return true;
        }
        
        return false;
    },
    
    /**
     * Create a comment thread (parent-child comments)
     */
    createCommentThread: (issueId: string, depth = 3) => {
        const comments: any[] = [];
        let parentId: string | undefined;
        
        for (let i = 0; i < depth; i++) {
            const comment = createCompleteComment({
                commentedOn: {
                    __typename: "Issue",
                    id: issueId,
                    isPublic: true,
                },
                parent: parentId ? { id: parentId } : undefined,
                owner: { id: `user_${i}_${"0".repeat(USER_ID_SUFFIX_LENGTH)}` },
            });
            
            comments.push(comment);
            parentId = comment.id;
        }
        
        return comments;
    },
    
    /**
     * Test comment voting permissions
     */
    canUserVoteOnComment: (userId: string, comment: any) => {
        // Cannot vote on own comments
        if (comment.owner?.id === userId) {
            return false;
        }
        
        // Must be able to read the comment to vote
        return commentPermissionHelpers.canUserReadComment(userId, comment);
    },
};

// Export complete test suite
export const commentPermissionTestSuite = commentPermissionFactory.createTestSuite();

/**
 * Usage examples:
 * 
 * ```typescript
 * import { commentScenarios, commentPermissionHelpers } from "./commentPermissions";
 * 
 * // Test commenting on different object types
 * const issueComment = commentPermissionHelpers.createCommentOn(
 *     "Issue",
 *     "issue_123",
 *     "user_456"
 * );
 * 
 * // Test nested comment threads
 * const thread = commentPermissionHelpers.createCommentThread("issue_123", 5);
 * 
 * // Test permission scenarios
 * const scenario = commentScenarios.publicIssueComment;
 * for (const actor of scenario.actors) {
 *     console.log(`${actor.id} can vote: ${actor.permissions.vote}`);
 * }
 * ```
 */
