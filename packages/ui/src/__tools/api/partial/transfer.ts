import { Transfer, TransferYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const transferYou: GqlPartial<TransferYou> = {
    __typename: "TransferYou",
    common: {
        canDelete: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};

export const transfer: GqlPartial<Transfer> = {
    __typename: "Transfer",
    common: {
        __define: {
            0: async () => rel((await import("./api")).api, "list"),
            1: async () => rel((await import("./note")).note, "list"),
            2: async () => rel((await import("./project")).project, "list"),
            3: async () => rel((await import("./routine")).routine, "list"),
            4: async () => rel((await import("./code")).code, "list"),
            5: async () => rel((await import("./standard")).standard, "list"),
        },
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
        fromOwner: async () => rel((await import("./user")).user, "nav"),
        toOwner: async () => rel((await import("./user")).user, "nav"),
        object: {
            __union: {
                Api: 0,
                Code: 4,
                Note: 1,
                Project: 2,
                Routine: 3,
                Standard: 5,
            },
        },
        you: () => rel(transferYou, "full"),
    },
    full: {},
    list: {},
};
