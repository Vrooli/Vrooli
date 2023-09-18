import { ChatInviteModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ChatInviteFormat: Formatter<ChatInviteModelLogic> = {
    gqlRelMap: {
        __typename: "ChatInvite",
        chat: "Chat",
        user: "User",
    },
    prismaRelMap: {
        __typename: "ChatInvite",
        chat: "Chat",
        user: "User",
    },
    joinMap: {},
    countFields: {},
};
