import { Note, NoteYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const noteYou: ApiPartial<NoteYou> = {
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

export const note: ApiPartial<Note> = {
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
        you: () => rel(noteYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./noteVersion.js")).noteVersion, "nav"),
        versions: async () => rel((await import("./noteVersion.js")).noteVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./noteVersion.js")).noteVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
