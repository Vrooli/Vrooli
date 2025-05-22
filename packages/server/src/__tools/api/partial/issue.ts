import { type Issue, type IssueTranslation, type IssueYou } from "@local/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const issueTranslation: ApiPartial<IssueTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const issueYou: ApiPartial<IssueYou> = {
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
};

export const issue: ApiPartial<Issue> = {
    common: {
        id: true,
        publicId: true,
        createdAt: true,
        updatedAt: true,
        closedAt: true,
        referencedVersionId: true,
        status: true,
        to: {
            __union: {
                Resource: async () => rel((await import("./resource.js")).resource, "nav"),
                Team: async () => rel((await import("./team.js")).team, "nav"),
            },
        },
        commentsCount: true,
        reportsCount: true,
        score: true,
        bookmarks: true,
        views: true,
        you: () => rel(issueYou, "full"),
    },
    full: {
        closedBy: async () => rel((await import("./user.js")).user, "nav"),
        createdBy: async () => rel((await import("./user.js")).user, "nav"),
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
