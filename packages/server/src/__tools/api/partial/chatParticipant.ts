import { ChatParticipant } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const chatParticipant: ApiPartial<ChatParticipant> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
    },
    list: {
        user: async () => rel((await import("./user")).user, "list"),
    },
};
