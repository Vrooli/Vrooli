import { SmartContract, SmartContractYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const smartContractYou: GqlPartial<SmartContractYou> = {
    __typename: "SmartContractYou",
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

export const smartContract: GqlPartial<SmartContract> = {
    __typename: "SmartContract",
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
        you: () => rel(smartContractYou, "full"),
    },
    full: {
        parent: async () => rel((await import("./smartContractVersion")).smartContractVersion, "nav"),
        versions: async () => rel((await import("./smartContractVersion")).smartContractVersion, "full", { omit: "root" }),
        stats: async () => rel((await import("./statsSmartContract")).statsSmartContract, "full"),
    },
    list: {
        versions: async () => rel((await import("./smartContractVersion")).smartContractVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isPrivate: true,
    },
};
