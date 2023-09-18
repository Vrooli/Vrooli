import { ChatParticipantModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ChatParticipantFormat: Formatter<ChatParticipantModelLogic> = {
    gqlRelMap: {
        __typename: "ChatParticipant",
        chat: "Chat",
        user: "User",
    },
    prismaRelMap: {
        __typename: "ChatParticipant",
        chat: "Chat",
        user: "User",
    },
    joinMap: {},
    countFields: {},
};
