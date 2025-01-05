import { Api, ApiYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const apiYou: GqlPartial<ApiYou> = {
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

export const api: GqlPartial<Api> = {
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
        you: () => rel(apiYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./apiVersion")).apiVersion, "nav"),
        versions: async () => rel((await import("./apiVersion")).apiVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./apiVersion")).apiVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
