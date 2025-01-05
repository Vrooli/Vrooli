import { Standard, StandardYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const standardYou: GqlPartial<StandardYou> = {
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

export const standard: GqlPartial<Standard> = {
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
        you: () => rel(standardYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./standardVersion")).standardVersion, "nav"),
        versions: async () => rel((await import("./standardVersion")).standardVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./standardVersion")).standardVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
