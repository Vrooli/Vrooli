import { Api, ApiYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const apiYou: ApiPartial<ApiYou> = {
    full: {
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

export const api: ApiPartial<Api> = {
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
        you: () => rel(apiYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./apiVersion.js")).apiVersion, "nav"),
        versions: async () => rel((await import("./apiVersion.js")).apiVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./apiVersion.js")).apiVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
