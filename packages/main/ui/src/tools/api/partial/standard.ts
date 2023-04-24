import { Standard, StandardYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const standardYou: GqlPartial<StandardYou> = {
    __typename: "StandardYou",
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

export const standard: GqlPartial<Standard> = {
    __typename: "Standard",
    common: {
        __define: {
            0: async () => rel((await import("./organization")).organization, "nav"),
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
                Organization: 0,
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
        you: () => rel(standardYou, "full"),
    },
    full: {
        parent: async () => rel((await import("./standardVersion")).standardVersion, "nav"),
        versions: async () => rel((await import("./standardVersion")).standardVersion, "full", { omit: "root" }),
        stats: async () => rel((await import("./statsStandard")).statsStandard, "full"),
    },
    list: {
        versions: async () => rel((await import("./standardVersion")).standardVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
