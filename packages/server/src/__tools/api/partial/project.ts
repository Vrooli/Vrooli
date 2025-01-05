import { Project, ProjectYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const projectYou: GqlPartial<ProjectYou> = {
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

export const project: GqlPartial<Project> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: async () => rel((await import("./label")).label, "list"),
        owner: {
            __union: {
                Team: async () => rel((await import("./team")).team, "nav"),
                User: async () => rel((await import("./user")).user, "nav"),
            },
        },
        permissions: true,
        questionsCount: true,
        score: true,
        bookmarks: true,
        tags: async () => rel((await import("./tag")).tag, "list"),
        transfersCount: true,
        views: true,
        you: () => rel(projectYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./projectVersion")).projectVersion, "nav"),
        versions: async () => rel((await import("./projectVersion")).projectVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./projectVersion")).projectVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
