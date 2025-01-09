import { Issue, IssueTranslation, IssueYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        created_at: true,
        updated_at: true,
        closedAt: true,
        referencedVersionId: true,
        status: true,
        to: {
            __union: {
                Api: async () => rel((await import("./api")).api, "nav"),
                Code: async () => rel((await import("./code")).code, "nav"),
                Note: async () => rel((await import("./note")).note, "nav"),
                Project: async () => rel((await import("./project")).project, "nav"),
                Routine: async () => rel((await import("./routine")).routine, "nav"),
                Standard: async () => rel((await import("./standard")).standard, "nav"),
                Team: async () => rel((await import("./team")).team, "nav"),
            },
        },
        commentsCount: true,
        reportsCount: true,
        score: true,
        bookmarks: true,
        views: true,
        labels: async () => rel((await import("./label")).label, "nav"),
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
