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

// Consistent test user IDs for permissions testing - using lazy initialization
let _testUserIds: Record<string, string> | null = null;
function getTestUserIds() {
    if (!_testUserIds) {
        _testUserIds = {
            commentOwner: generatePK().toString(),
            issueOwner: generatePK().toString(),
            otherUser: generatePK().toString(),
            teamMember: generatePK().toString(),
            resourceOwner: generatePK().toString(),
            threadStarter: generatePK().toString(),
            replyAuthor: generatePK().toString(),
        };
    }
    return _testUserIds;
}

// Minimal comment object for testing
function createMinimalComment(overrides: Record<string, unknown> = {}) {
    return {
        __typename: "Comment",
        id: generatePK().toString(),
        owner: {
            __typename: "User",
            id: getTestUserIds().commentOwner,
        },
        commentedOn: {
            __typename: "Issue",
            id: DUMMY_ID,
        },
        translations: [{
            __typename: "CommentTranslation",
            id: generatePK().toString(),
            language: "en",
            text: "Test comment",
        }],
        ...overrides,
    };
}

// Complete comment object for testing
function createCompleteComment(overrides: Record<string, unknown> = {}) {
    return {
        __typename: "Comment",
        id: generatePK().toString(),
        owner: {
            __typename: "User",
            id: getTestUserIds().commentOwner,
            handle: "commenter",
            name: "Test Commenter",
        },
        commentedOn: {
            __typename: "Issue",
            id: DUMMY_ID,
            title: "Test Issue",
            isPublic: true,
            owner: { id: getTestUserIds().issueOwner },
        },
        translations: [{
            __typename: "CommentTranslation",
            id: generatePK().toString(),
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
                const membership = session._testTeamMembership as { teamId: string; role: string };
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
                const membership = session._testTeamMembership as { teamId: string; role: string };
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
            return commentPermissionFactory.config.customRules?.read?.(session, comment) ?? false;
        },
        
        // Custom rule: anyone can vote on comments they can read
        vote: (session, comment) => {
            // Must be able to read the comment to vote on it
            // Cannot vote on your own comments
            if ("id" in session && comment.owner?.id === session.id) {
                return false;
            }
            
            return commentPermissionFactory.config.customRules?.read?.(session, comment) ?? false;
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
                owner: { id: getTestUserIds().issueOwner },
            },
        }),
        actors: [
            {
                id: "comment_owner",
                session: { id: getTestUserIds().commentOwner } as Record<string, unknown>,
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
                session: { id: getTestUserIds().issueOwner } as Record<string, unknown>,
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
                session: { id: getTestUserIds().otherUser } as Record<string, unknown>,
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
                owner: { id: getTestUserIds().resourceOwner },
            },
        }),
        actors: [
            {
                id: "comment_owner",
                session: { id: getTestUserIds().commentOwner } as Record<string, unknown>,
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
                session: { id: getTestUserIds().resourceOwner } as Record<string, unknown>,
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
            owner: {
                __typename: "User",
                id: getTestUserIds().teamMember,
                handle: "teammember",
                name: "Team Member",
            },
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
                    id: getTestUserIds().teamMember,
                    _testTeamMembership: { teamId: "team_123", role: "Member" },
                } as Record<string, unknown>,
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
                session: { id: getTestUserIds().otherUser } as Record<string, unknown>,
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
            owner: {
                __typename: "User",
                id: getTestUserIds().replyAuthor,
                handle: "replyauthor",
                name: "Reply Author",
            },
            parent: {
                __typename: "Comment",
                id: "parent_comment_id",
                owner: { id: getTestUserIds().threadStarter },
            },
        }),
        actors: [
            {
                id: "thread_starter",
                session: { id: getTestUserIds().threadStarter } as Record<string, unknown>,
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
                session: { id: getTestUserIds().replyAuthor } as Record<string, unknown>,
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
    canUserCommentOn: (userId: string, targetObject: Record<string, unknown>) => {
        // Can comment on public objects
        if (targetObject.isPublic) {
            return true;
        }
        
        // Can comment on own objects
        if (targetObject.owner?.id === userId) {
            return true;
        }
        
        // Can comment on team objects if team member
        if (targetObject.team && targetObject.team.members?.some((m: Record<string, unknown>) => m.userId === userId)) {
            return true;
        }
        
        return false;
    },
    
    /**
     * Test if user can read a comment
     */
    canUserReadComment: (userId: string, comment: Record<string, unknown>) => {
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
        const comments: Record<string, unknown>[] = [];
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
            parentId = comment.id.toString();
        }
        
        return comments;
    },
    
    /**
     * Test comment voting permissions
     */
    canUserVoteOnComment: (userId: string, comment: Record<string, unknown>) => {
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
