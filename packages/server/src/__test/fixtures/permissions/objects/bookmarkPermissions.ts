/**
 * Bookmark Permission Fixtures
 * 
 * Comprehensive permission testing fixtures for Bookmark objects.
 */

import { generatePK, DUMMY_ID } from "@vrooli/shared";
import { ObjectPermissionFactory } from "../factories/ObjectPermissionFactory.js";

// Consistent test user IDs for permissions testing - using lazy initialization
let _testUserIds: Record<string, string> | null = null;
function getTestUserIds() {
    if (!_testUserIds) {
        _testUserIds = {
            bookmarkOwner: generatePK().toString(),
            projectOwner: generatePK().toString(),
            otherUser: generatePK().toString(),
            admin: generatePK().toString(),
        };
    }
    return _testUserIds;
}

// Minimal bookmark object for testing
function createMinimalBookmark(overrides: Record<string, unknown> = {}) {
    return {
        __typename: "Bookmark",
        id: generatePK().toString(),
        to: {
            __typename: "Project",
            id: DUMMY_ID,
        },
        list: {
            __typename: "BookmarkList", 
            id: generatePK().toString(),
        },
        ...overrides,
    };
}

// Complete bookmark object for testing
function createCompleteBookmark(overrides: Record<string, unknown> = {}) {
    return {
        __typename: "Bookmark",
        id: generatePK().toString(),
        to: {
            __typename: "Project",
            id: DUMMY_ID,
            name: "Test Project",
            isPublic: true,
        },
        list: {
            __typename: "BookmarkList",
            id: generatePK().toString(),
            label: "My Bookmarks",
            owner: { id: getTestUserIds().bookmarkOwner },
            isPrivate: false,
        },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        ...overrides,
    };
}

// Create bookmark permission factory
export const bookmarkPermissionFactory = new ObjectPermissionFactory({
    objectType: "Bookmark",
    createMinimal: createMinimalBookmark,
    createComplete: createCompleteBookmark,
    supportedActions: ["read", "create", "update", "delete"],
    canBeTeamOwned: false, // Bookmarks are user-owned
    hasVisibility: false, // Bookmarks don't have visibility settings
    customRules: {
        // Custom rule: users can only bookmark public objects or their own objects
        create: (session, bookmark) => {
            const targetObject = bookmark.to;
            
            // If target is public, anyone can bookmark it
            if (targetObject.isPublic) {
                return true;
            }
            
            // If target is owned by the user, they can bookmark it
            if ("id" in session && targetObject.owner?.id === session.id) {
                return true;
            }
            
            return false;
        },
        
        // Custom rule: users can only access bookmarks in their own lists
        read: (session, bookmark) => {
            if ("id" in session && bookmark.list?.owner?.id === session.id) {
                return true;
            }
            
            // Admins can see all bookmarks
            if ("roles" in session && session.roles?.some(r => r.role.name === "Admin")) {
                return true;
            }
            
            return false;
        },
    },
});

// Pre-configured bookmark scenarios
export const bookmarkScenarios = {
    /**
     * User bookmarking a public project
     */
    publicProjectBookmark: bookmarkPermissionFactory.createPublicUserOwned(),
    
    /**
     * User bookmarking their own private project
     */
    ownPrivateProjectBookmark: {
        ...bookmarkPermissionFactory.createPrivateUserOwned(),
        id: "bookmark_own_private_project",
        description: "User bookmarking their own private project",
    },
    
    /**
     * Attempting to bookmark another user's private project (should fail)
     */
    otherPrivateProjectBookmark: {
        id: "bookmark_other_private_project",
        description: "Attempting to bookmark another user's private project",
        resource: createCompleteBookmark({
            to: {
                __typename: "Project",
                id: DUMMY_ID,
                name: "Private Project",
                isPublic: false,
                owner: { id: getTestUserIds().projectOwner }, // Different owner
            },
            list: {
                owner: { id: getTestUserIds().bookmarkOwner }, // User's list
            },
        }),
        actors: [
            {
                id: "user",
                session: { id: getTestUserIds().bookmarkOwner } as Record<string, unknown>,
                permissions: {
                    read: false,
                    create: false, // Cannot bookmark private object they don't own
                    update: false,
                    delete: false,
                },
            },
        ],
        actions: ["read", "create", "update", "delete"],
    },
    
    /**
     * Cross-user bookmark list access (should fail)
     */
    crossUserBookmarkAccess: {
        id: "bookmark_cross_user_access",
        description: "Attempting to access another user's bookmark list",
        resource: createCompleteBookmark({
            list: {
                owner: { id: getTestUserIds().otherUser }, // Different user's list
            },
        }),
        actors: [
            {
                id: "user",
                session: { id: getTestUserIds().bookmarkOwner } as Record<string, unknown>,
                permissions: {
                    read: false, // Cannot read other user's bookmark
                    create: false,
                    update: false,
                    delete: false,
                },
            },
            {
                id: "admin",
                session: { 
                    id: getTestUserIds().admin,
                    roles: [{ role: { name: "Admin", permissions: "[\"*\"]" } }],
                } as Record<string, unknown>,
                permissions: {
                    read: true, // Admins can see all bookmarks
                    create: true,
                    update: true,
                    delete: true,
                },
            },
        ],
        actions: ["read", "create", "update", "delete"],
    },
};

// Helper functions for testing bookmark permissions
export const bookmarkPermissionHelpers = {
    /**
     * Create a bookmark in a user's list
     */
    createUserBookmark: (userId: string, targetObjectId: string, listId?: string) => {
        return createCompleteBookmark({
            to: { id: targetObjectId },
            list: {
                id: listId || generatePK().toString(),
                owner: { id: userId },
            },
        });
    },
    
    /**
     * Create a bookmark list owned by a user
     */
    createUserBookmarkList: (userId: string, isPrivate = false) => {
        return {
            __typename: "BookmarkList",
            id: generatePK().toString(),
            label: "Test Bookmark List",
            owner: { id: userId },
            isPrivate,
            bookmarks: [],
        };
    },
    
    /**
     * Test if user can bookmark a specific object
     */
    canUserBookmarkObject: (userId: string, targetObject: Record<string, unknown>) => {
        // Public objects can be bookmarked by anyone
        if (targetObject.isPublic) {
            return true;
        }
        
        // Users can bookmark their own objects
        if (targetObject.owner?.id === userId) {
            return true;
        }
        
        return false;
    },
    
    /**
     * Test if user can access a bookmark
     */
    canUserAccessBookmark: (userId: string, bookmark: Record<string, unknown>) => {
        // Users can access bookmarks in their own lists
        if (bookmark.list?.owner?.id === userId) {
            return true;
        }
        
        return false;
    },
};

// Export complete test suite
export const bookmarkPermissionTestSuite = bookmarkPermissionFactory.createTestSuite();

/**
 * Usage examples:
 * 
 * ```typescript
 * import { bookmarkScenarios, bookmarkPermissionHelpers } from "./bookmarkPermissions";
 * 
 * // Test basic bookmark creation
 * const scenario = bookmarkScenarios.publicProjectBookmark;
 * for (const actor of scenario.actors) {
 *     const canCreate = actor.permissions.create;
 *     // Test create permission
 * }
 * 
 * // Test custom bookmark scenarios
 * const userBookmark = bookmarkPermissionHelpers.createUserBookmark(
 *     "222222222222222222",
 *     "project_123"
 * );
 * 
 * const canAccess = bookmarkPermissionHelpers.canUserAccessBookmark(
 *     "222222222222222222",
 *     userBookmark
 * );
 * ```
 */
