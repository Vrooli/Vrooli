import { Formatter } from "../types";

const __typename = "ChatInvite" as const;
export const ChatInviteFormat: Formatter<ModelChatInviteLogic> = {
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
