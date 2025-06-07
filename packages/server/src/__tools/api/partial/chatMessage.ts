import { type ChatMessage, type ChatMessageSearchTreeResult, type ChatMessageYou } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const chatMessageYou: ApiPartial<ChatMessageYou> = {
    common: {
        canDelete: true,
        canReply: true,
        canReport: true,
        canUpdate: true,
        canReact: true,
        reaction: true,
    },
};

export const chatMessage: ApiPartial<ChatMessage> = {
    common: {
        id: true,
        createdAt: true,
        sequence: true,
        versionIndex: true,
        parent: { id: true, createdAt: true },
        user: async () => rel((await import("./user.js")).user, "nav"),
        score: true,
        reactionSummaries: async () => rel((await import("./reaction.js")).reactionSummary, "list"),
        reportsCount: true,
        you: () => rel(chatMessageYou, "full"),
    },
    full: {
        config: true,
        text: true,
    },
    list: {
        config: true,
        text: true,
    },
};

export const chatMessageSearchTreeResult: ApiPartial<ChatMessageSearchTreeResult> = {
    common: {
        hasMoreUp: true,
        hasMoreDown: true,
        messages: () => rel(chatMessage, "list"),
    },
};
