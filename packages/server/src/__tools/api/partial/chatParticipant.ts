import { ChatParticipant } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const chatParticipant: ApiPartial<ChatParticipant> = {
    common: {
        id: true,
        createdAt: true,
        updatedAt: true,
    },
    list: {
        user: async () => rel((await import("./user.js")).user, "list"),
    },
};
