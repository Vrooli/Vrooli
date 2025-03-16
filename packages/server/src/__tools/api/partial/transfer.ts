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
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
        fromOwner: async () => rel((await import("./user.js")).user, "nav"),
        toOwner: async () => rel((await import("./user.js")).user, "nav"),
        object: {
            __union: {
                Api: async () => rel((await import("./api.js")).api, "list"),
                Code: async () => rel((await import("./code.js")).code, "list"),
                Note: async () => rel((await import("./note.js")).note, "list"),
                Project: async () => rel((await import("./project.js")).project, "list"),
                Routine: async () => rel((await import("./routine.js")).routine, "list"),
                Standard: async () => rel((await import("./standard.js")).standard, "list"),
            },
        },
        you: () => rel(transferYou, "full"),
    },
};
