import { ChatParticipantModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ChatParticipant" as const;
export const ChatParticipantFormat: Formatter<ChatParticipantModelLogic> = {
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
