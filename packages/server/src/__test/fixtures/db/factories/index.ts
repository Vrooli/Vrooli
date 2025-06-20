/**
 * Content & Interaction System Database Factories
 * 
 * These factories handle polymorphic relationships and engagement features:
 * - Comments: Threaded discussions on any object
 * - Bookmarks: User collections of any object
 * - Views: Analytics and tracking (anonymous/authenticated)
 * - Reactions: Likes, dislikes, and emoji reactions
 * - ReactionSummaries: Aggregated reaction counts
 */

export { BookmarkListDbFactory, createBookmarkListDbFactory } from './BookmarkListDbFactory.js';
export { CommentDbFactory, createCommentDbFactory } from './CommentDbFactory.js';
export { BookmarkDbFactory, createBookmarkDbFactory } from './BookmarkDbFactory.js';
export { ViewDbFactory, createViewDbFactory } from './ViewDbFactory.js';
export { ReactionDbFactory, createReactionDbFactory } from './ReactionDbFactory.js';
export { ReactionSummaryDbFactory, createReactionSummaryDbFactory } from './ReactionSummaryDbFactory.js';

// Re-export from existing factories
export { UserDbFactory, createUserDbFactory } from './UserDbFactory.js';
export { ChatDbFactory, createChatDbFactory } from './ChatDbFactory.js';