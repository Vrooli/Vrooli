import { Comment, CommentThread, CommentTranslation, CommentYou } from "@local/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const commentTranslation: GqlPartial<CommentTranslation> = {
    __typename: "CommentTranslation",
    common: {
        id: true,
        language: true,
        text: true,
    },
    full: {},
    list: {},
};

export const commentYou: GqlPartial<CommentYou> = {
    __typename: "CommentYou",
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
    full: {},
    list: {},
};

export const comment: GqlPartial<Comment> = {
    __typename: "Comment",
    common: {
        __define: {
            0: async () => rel((await import("./organization")).organization, "nav"),
            1: async () => rel((await import("./user")).user, "nav"),
        },
        id: true,
        created_at: true,
        updated_at: true,
        owner: {
            __union: {
                Organization: 0,
                User: 1,
            },
        },
        score: true,
        bookmarks: true,
        reportsCount: true,
        you: () => rel(commentYou, "full"),
    },
    full: {
        __define: {
            0: async () => rel((await import("./apiVersion")).apiVersion, "nav"),
            1: async () => rel((await import("./issue")).issue, "nav"),
            2: async () => rel((await import("./noteVersion")).noteVersion, "nav"),
            3: async () => rel((await import("./post")).post, "nav"),
            4: async () => rel((await import("./projectVersion")).projectVersion, "nav"),
            5: async () => rel((await import("./pullRequest")).pullRequest, "nav"),
            6: async () => rel((await import("./question")).question, "nav"),
            7: async () => rel((await import("./questionAnswer")).questionAnswer, "nav"),
            8: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
            9: async () => rel((await import("./smartContractVersion")).smartContractVersion, "nav"),
            10: async () => rel((await import("./standardVersion")).standardVersion, "nav"),
            11: async () => rel((await import("./organization")).organization, "nav"),
            12: async () => rel((await import("./user")).user, "nav"),
        },
        commentedOn: {
            __union: {
                ApiVersion: 0,
                Issue: 1,
                NoteVersion: 2,
                Post: 3,
                ProjectVersion: 4,
                PullRequest: 5,
                Question: 6,
                QuestionAnswer: 7,
                RoutineVersion: 8,
                SmartContractVersion: 9,
                StandardVersion: 10,
            },
        },
        translations: () => rel(commentTranslation, "full"),
    },
    list: {
        translations: () => rel(commentTranslation, "list"),
    },
};

export const commentThread: GqlPartial<CommentThread> = {
    __typename: "CommentThread",
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
