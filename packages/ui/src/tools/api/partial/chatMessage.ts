import { AutoFillResult, ChatMessage, ChatMessageSearchTreeResult, ChatMessageTranslation, ChatMessageYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const chatMessageTranslation: GqlPartial<ChatMessageTranslation> = {
    __typename: "ChatMessageTranslation",
    common: {
        id: true,
        language: true,
        text: true,
    },
    full: {},
    list: {},
};

export const chatMessageYou: GqlPartial<ChatMessageYou> = {
    __typename: "ChatMessageYou",
    common: {
        canDelete: true,
        canReply: true,
        canReport: true,
        canUpdate: true,
        canReact: true,
        reaction: true,
    },
    full: {},
    list: {},
};

export const chatMessage: GqlPartial<ChatMessage> = {
    __typename: "ChatMessage",
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
        chat: async () => rel((await import("./chat")).chat, "full", { omit: "messages" }),
        translations: () => rel(chatMessageTranslation, "full"),
    },
    list: {
        translations: () => rel(chatMessageTranslation, "list"),
    },
};

export const chatMessageSearchTreeResult: GqlPartial<ChatMessageSearchTreeResult> = {
    __typename: "ChatMessageSearchTreeResult",
    common: {
        hasMoreUp: true,
        hasMoreDown: true,
        messages: () => rel(chatMessage, "list"),
    },
};

export const autoFillResult: GqlPartial<AutoFillResult> = {
    __typename: "AutoFillResult",
    common: {
        data: true,
    },
};
