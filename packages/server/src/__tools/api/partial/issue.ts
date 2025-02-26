import { Issue, IssueTranslation, IssueYou } from "@local/shared";
import { ApiPartial } from "../types.js";
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
        created_at: true,
        updated_at: true,
        closedAt: true,
        referencedVersionId: true,
        status: true,
        to: {
            __union: {
                Api: async () => rel((await import("./api.js")).api, "nav"),
                Code: async () => rel((await import("./code.js")).code, "nav"),
                Note: async () => rel((await import("./note.js")).note, "nav"),
                Project: async () => rel((await import("./project.js")).project, "nav"),
                Routine: async () => rel((await import("./routine.js")).routine, "nav"),
                Standard: async () => rel((await import("./standard.js")).standard, "nav"),
                Team: async () => rel((await import("./team.js")).team, "nav"),
            },
        },
        commentsCount: true,
        reportsCount: true,
        score: true,
        bookmarks: true,
        views: true,
        labels: async () => rel((await import("./label.js")).label, "nav"),
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
