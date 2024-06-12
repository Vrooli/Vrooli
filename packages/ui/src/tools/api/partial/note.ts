import { Note, NoteYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const noteYou: GqlPartial<NoteYou> = {
    __typename: "NoteYou",
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

export const note: GqlPartial<Note> = {
    __typename: "Note",
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
        you: () => rel(noteYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./noteVersion")).noteVersion, "nav"),
        versions: async () => rel((await import("./noteVersion")).noteVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./noteVersion")).noteVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
