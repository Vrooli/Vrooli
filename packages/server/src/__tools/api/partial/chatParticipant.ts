import { ChatParticipant } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const chatParticipant: GqlPartial<ChatParticipant> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
    },
    list: {
        user: async () => rel((await import("./user")).user, "list"),
    },
};
