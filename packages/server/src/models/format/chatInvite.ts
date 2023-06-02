import { ChatInviteModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ChatInvite" as const;
export const ChatInviteFormat: Formatter<ChatInviteModelLogic> = {
    gqlRelMap: {
        __typename,
        chat: "Chat",
        user: "User",
    },
    prismaRelMap: {
        __typename,
        chat: "Chat",
        user: "User",
    },
    joinMap: {},
    countFields: {},
};
