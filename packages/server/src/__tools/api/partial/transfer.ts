import { type Transfer, type TransferYou } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
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
        closedAt: true,
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
