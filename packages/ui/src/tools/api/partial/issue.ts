import { Issue, IssueTranslation, IssueYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const issueTranslation: GqlPartial<IssueTranslation> = {
    __typename: "IssueTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};

export const issueYou: GqlPartial<IssueYou> = {
    __typename: "IssueYou",
    common: {
        canComment: true,
        canDelete: true,
        canBookmark: true,
        canReport: true,
        canUpdate: true,
        canRead: true,
        canReact: true,
        isBookmarked: true,
        reaction: true,
    },
    full: {},
    list: {},
};

export const issue: GqlPartial<Issue> = {
    __typename: "Issue",
    common: {
        __define: {
            0: async () => rel((await import("./api")).api, "nav"),
            1: async () => rel((await import("./note")).note, "nav"),
            2: async () => rel((await import("./organization")).organization, "nav"),
            3: async () => rel((await import("./project")).project, "nav"),
            4: async () => rel((await import("./routine")).routine, "nav"),
            5: async () => rel((await import("./smartContract")).smartContract, "nav"),
            6: async () => rel((await import("./standard")).standard, "nav"),
            7: async () => rel((await import("./label")).label, "nav"),
        },
        id: true,
        created_at: true,
        updated_at: true,
        closedAt: true,
        referencedVersionId: true,
        status: true,
        to: {
            __union: {
                Api: 0,
                Note: 1,
                Organization: 2,
                Project: 3,
                Routine: 4,
                SmartContract: 5,
                Standard: 6,
            },
        },
        commentsCount: true,
        reportsCount: true,
        score: true,
        bookmarks: true,
        views: true,
        labels: { __use: 7 },
        you: () => rel(issueYou, "full"),
    },
    full: {
        closedBy: async () => rel((await import("./user")).user, "nav"),
        createdBy: async () => rel((await import("./user")).user, "nav"),
        translations: () => rel(issueTranslation, "full"),
    },
    list: {
        translations: () => rel(issueTranslation, "list"),
    },
    nav: {
        id: true,
        translations: () => rel(issueTranslation, "list"),
    },
};
