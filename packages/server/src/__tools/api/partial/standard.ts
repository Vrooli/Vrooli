import { Standard, StandardYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const standardYou: ApiPartial<StandardYou> = {
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

export const standard: ApiPartial<Standard> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: async () => rel((await import("./label.js")).label, "list"),
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
        you: () => rel(standardYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./standardVersion.js")).standardVersion, "nav"),
        versions: async () => rel((await import("./standardVersion.js")).standardVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./standardVersion.js")).standardVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
