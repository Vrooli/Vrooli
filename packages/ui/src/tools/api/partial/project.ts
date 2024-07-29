import { Project, ProjectYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const projectYou: GqlPartial<ProjectYou> = {
    __typename: "ProjectYou",
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
    full: {},
    list: {},
};

export const project: GqlPartial<Project> = {
    __typename: "Project",
    common: {
        __define: {
            0: async () => rel((await import("./team")).team, "nav"),
            1: async () => rel((await import("./user")).user, "nav"),
            2: async () => rel((await import("./tag")).tag, "list"),
            3: async () => rel((await import("./label")).label, "list"),
        },
        id: true,
        created_at: true,
        updated_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: { __use: 3 },
        owner: {
            __union: {
                Team: 0,
                User: 1,
            },
        },
        permissions: true,
        questionsCount: true,
        score: true,
        bookmarks: true,
        tags: { __use: 2 },
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
