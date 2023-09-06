import { ChatMessageModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ChatMessage" as const;
export const ChatMessageFormat: Formatter<ChatMessageModelLogic> = {
    gqlRelMap: {
        __typename,
        chat: "Chat",
        user: "User",
        reactionSummaries: "ReactionSummary",
        reports: "Report",
    },
    prismaRelMap: {
        __typename,
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
