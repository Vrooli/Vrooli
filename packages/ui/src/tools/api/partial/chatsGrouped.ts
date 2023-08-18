import { ChatsGrouped } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const chatsGrouped: GqlPartial<ChatsGrouped> = {
    __typename: "ChatsGrouped",
    common: {
        chatsCount: true,
        participants: async () => rel((await import("./chatParticipant")).chatParticipant, "list", { omit: "chat" }),
    },
};
