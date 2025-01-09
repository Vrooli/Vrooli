import { Comment, CommentSearchResult, CommentThread, CommentTranslation, CommentYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        created_at: true,
        updated_at: true,
        owner: {
            __union: {
                Team: async () => rel((await import("./team")).team, "nav"),
                User: async () => rel((await import("./user")).user, "nav"),
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
                ApiVersion: async () => rel((await import("./apiVersion")).apiVersion, "nav"),
                CodeVersion: async () => rel((await import("./codeVersion")).codeVersion, "nav"),
                Issue: async () => rel((await import("./issue")).issue, "nav"),
                NoteVersion: async () => rel((await import("./noteVersion")).noteVersion, "nav"),
                Post: async () => rel((await import("./post")).post, "nav"),
                ProjectVersion: async () => rel((await import("./projectVersion")).projectVersion, "nav"),
                PullRequest: async () => rel((await import("./pullRequest")).pullRequest, "nav"),
                Question: async () => rel((await import("./question")).question, "common"),
                QuestionAnswer: async () => rel((await import("./questionAnswer")).questionAnswer, "common"),
                RoutineVersion: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
                StandardVersion: async () => rel((await import("./standardVersion")).standardVersion, "nav"),
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
