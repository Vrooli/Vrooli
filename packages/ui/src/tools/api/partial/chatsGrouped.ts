import { ChatsGrouped } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const chatsGrouped: GqlPartial<ChatsGrouped> = {
    __typename: "ChatsGrouped",
    common: {
        id: true,
        chatsCount: true,
        user: async () => rel((await import("./user")).user, "nav"),
    },
};
