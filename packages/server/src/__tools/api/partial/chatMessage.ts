import { ChatMessage, ChatMessageSearchTreeResult, ChatMessageTranslation, ChatMessageYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const chatMessageTranslation: ApiPartial<ChatMessageTranslation> = {
    common: {
        id: true,
        language: true,
        text: true,
    },
};

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
        created_at: true,
        updated_at: true,
        sequence: true,
        versionIndex: true,
        parent: { id: true, created_at: true },
        user: async () => rel((await import("./user")).user, "nav"),
        score: true,
        reactionSummaries: async () => rel((await import("./reaction")).reactionSummary, "list"),
        reportsCount: true,
        you: () => rel(chatMessageYou, "full"),
    },
    full: {
        translations: () => rel(chatMessageTranslation, "full"),
    },
    list: {
        translations: () => rel(chatMessageTranslation, "list"),
    },
};

export const chatMessageSearchTreeResult: ApiPartial<ChatMessageSearchTreeResult> = {
    common: {
        hasMoreUp: true,
        hasMoreDown: true,
        messages: () => rel(chatMessage, "list"),
    },
};