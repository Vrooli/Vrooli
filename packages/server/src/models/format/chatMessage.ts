import { ChatMessageModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ChatMessageFormat: Formatter<ChatMessageModelLogic> = {
    gqlRelMap: {
        __typename: "ChatMessage",
        chat: "Chat",
        user: "User",
        reactionSummaries: "ReactionSummary",
        reports: "Report",
    },
    prismaRelMap: {
        __typename: "ChatMessage",
        chat: "Chat",
        fork: "ChatMessage",
        children: "ChatMessage",
        user: "User",
        reactionSummaries: "ReactionSummary",
        reports: "Report",
    },
    joinMap: {},
    countFields: {
        reportsCount: true,
        translationsCount: true,
    },
};
