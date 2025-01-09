import { Transfer, TransferYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const transferYou: ApiPartial<TransferYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};

export const transfer: ApiPartial<Transfer> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
        fromOwner: async () => rel((await import("./user")).user, "nav"),
        toOwner: async () => rel((await import("./user")).user, "nav"),
        object: {
            __union: {
                Api: async () => rel((await import("./api")).api, "list"),
                Code: async () => rel((await import("./code")).code, "list"),
                Note: async () => rel((await import("./note")).note, "list"),
                Project: async () => rel((await import("./project")).project, "list"),
                Routine: async () => rel((await import("./routine")).routine, "list"),
                Standard: async () => rel((await import("./standard")).standard, "list"),
            },
        },
        you: () => rel(transferYou, "full"),
    },
};
