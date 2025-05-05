import { Transfer, TransferYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const transferYou: ApiPartial<TransferYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};

export const transfer: ApiPartial<Transfer> = {
    common: {
        id: true,
        createdAt: true,
        updatedAt: true,
        mergedOrRejectedAt: true,
        status: true,
        fromOwner: async () => rel((await import("./user.js")).user, "nav"),
        toOwner: async () => rel((await import("./user.js")).user, "nav"),
        object: {
            __union: {
                Resource: async () => rel((await import("./resource.js")).resource, "list"),
            },
        },
        you: () => rel(transferYou, "full"),
    },
};
