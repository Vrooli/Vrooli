import { Project, ProjectYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const projectYou: ApiPartial<ProjectYou> = {
    common: {
        canDelete: true,
        canBookmark: true,
        canTransfer: true,
        canUpdate: true,
        canRead: true,
        canReact: true,
        isBookmarked: true,
        isViewed: true,
        reaction: true,
    },
};

export const project: ApiPartial<Project> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isPrivate: true,
        issuesCount: true,
        owner: {
            __union: {
                Team: async () => rel((await import("./team.js")).team, "nav"),
                User: async () => rel((await import("./user.js")).user, "nav"),
            },
        },
        permissions: true,
        score: true,
        bookmarks: true,
        tags: async () => rel((await import("./tag.js")).tag, "list"),
        transfersCount: true,
        views: true,
        you: () => rel(projectYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./projectVersion.js")).projectVersion, "nav"),
        versions: async () => rel((await import("./projectVersion.js")).projectVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./projectVersion.js")).projectVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
