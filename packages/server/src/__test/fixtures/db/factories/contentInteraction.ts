/**
 * Content & Interaction Object Factories
 * 
 * This module exports database fixture factories for content and interaction objects.
 * These objects support polymorphic relationships and can be attached to various entity types.
 */

// Re-export all content & interaction factories
export { 
    CommentDbFactory, 
    commentDbFixtures, 
    commentDbIds,
    seedCommentThread,
    seedTestComments,
} from "../commentFixtures.js";

export { 
    TagDbFactory, 
    tagDbFixtures,
    tagDbIds,
    seedTags,
    seedTagHierarchy,
    seedPopularTags,
} from "../tagFixtures.js";

export { 
    BookmarkDbFactory,
    BookmarkListDbFactory,
    bookmarkDbFixtures,
    bookmarkListDbFixtures,
    bookmarkDbIds,
    seedBookmarks,
} from "../bookmarkFixtures.js";

export { 
    ViewDbFactory,
    viewDbFixtures,
    viewDbIds,
    seedViews,
    seedRecentActivity,
    seedViewAnalytics,
    seedViewsByDateRange,
} from "../viewFixtures.js";

// Export types
export type { DbTestFixtures, BulkSeedOptions, BulkSeedResult } from "../types.js";

/**
 * Quick reference for polymorphic relationships:
 * 
 * Comments:
 * - CommentFor enum: Issue, PullRequest, ResourceVersion
 * - Use CommentDbFactory.createForObject() with CommentFor enum values
 * 
 * Bookmarks:
 * - BookmarkFor enum: Comment, Issue, Resource, Tag, Team, User
 * - Additional types: Api, Code, Note, Post, Project, Prompt, Question, Quiz, Routine, etc.
 * - Use BookmarkDbFactory.createForObject() with any supported type
 * 
 * Tags:
 * - Tags are reusable across all objects through many-to-many relationships
 * - Support hierarchical structure with parent-child relationships
 * 
 * Views:
 * - ViewFor enum: Resource, ResourceVersion, Team, User
 * - Legacy support for "Issue" type
 * - Support both authenticated (byId) and anonymous (bySessionId) views
 */