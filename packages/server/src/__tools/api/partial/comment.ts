import { Comment, CommentSearchResult, CommentThread, CommentTranslation, CommentYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const commentTranslation: ApiPartial<CommentTranslation> = {
    common: {
        id: true,
        language: true,
        text: true,
    },
};

export const commentYou: ApiPartial<CommentYou> = {
    common: {
        canDelete: true,
        canBookmark: true,
        canReply: true,
        canReport: true,
        canUpdate: true,
        canReact: true,
        isBookmarked: true,
        reaction: true,
    },
};

export const comment: ApiPartial<Comment> = {
    common: {
        id: true,
        createdAt: true,
        updatedAt: true,
        owner: {
            __union: {
                Team: async () => rel((await import("./team.js")).team, "nav"),
                User: async () => rel((await import("./user.js")).user, "nav"),
            },
        },
        score: true,
        bookmarks: true,
        reportsCount: true,
        you: () => rel(commentYou, "full"),
    },
    full: {
        commentedOn: {
            __union: {
                Issue: async () => rel((await import("./issue.js")).issue, "nav"),
                PullRequest: async () => rel((await import("./pullRequest.js")).pullRequest, "nav"),
                ResourceVersion: async () => rel((await import("./resourceVersion.js")).resourceVersion, "nav"),
            },
        },
        translations: () => rel(commentTranslation, "full"),
    },
    list: {
        translations: () => rel(commentTranslation, "list"),
    },
};

export const commentThread: ApiPartial<CommentThread> = {
    common: {
        childThreads: {
            childThreads: {
                comment: () => rel(comment, "list"),
                endCursor: true,
                totalInThread: true,
            },
            comment: () => rel(comment, "list"),
            endCursor: true,
            totalInThread: true,
        },
        comment: () => rel(comment, "list"),
        endCursor: true,
        totalInThread: true,
    },
};

export const commentSearchResult: ApiPartial<CommentSearchResult> = {
    common: {
        endCursor: true,
        threads: () => rel(commentThread, "common"),
        totalThreads: true,
    },
};
