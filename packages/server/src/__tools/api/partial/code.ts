import { Code, CodeYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const codeYou: ApiPartial<CodeYou> = {
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

export const code: ApiPartial<Code> = {
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
        you: () => rel(codeYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./codeVersion.js")).codeVersion, "nav"),
        versions: async () => rel((await import("./codeVersion.js")).codeVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./codeVersion.js")).codeVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
