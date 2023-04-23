import { rel } from "../utils";
export const chatMessageTranslation = {
    __typename: "ChatMessageTranslation",
    common: {
        id: true,
        language: true,
        text: true,
    },
    full: {},
    list: {},
};
export const chatMessageYou = {
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
export const chatMessage = {
    __typename: "ChatMessage",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        user: async () => rel((await import("./user")).user, "nav"),
        score: true,
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
//# sourceMappingURL=chatMessage.js.map