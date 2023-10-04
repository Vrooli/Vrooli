import { Chat, ChatTranslation, ChatYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const chatTranslation: GqlPartial<ChatTranslation> = {
    __typename: "ChatTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};

export const chatYou: GqlPartial<ChatYou> = {
    __typename: "ChatYou",
    common: {
        canDelete: true,
        canInvite: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};

export const chat: GqlPartial<Chat> = {
    __typename: "Chat",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        openToAnyoneWithInvite: true,
        organization: async () => rel((await import("./organization")).organization, "nav"),
        restrictedToRoles: async () => rel((await import("./role")).role, "full"),
        participants: async () => rel((await import("./chatParticipant")).chatParticipant, "list", { omit: "chat" }),
        participantsCount: true,
        invitesCount: true,
        you: () => rel(chatYou, "full"),
    },
    full: {
        __define: {
            0: async () => rel((await import("./label")).label, "full"),
        },
        participants: async () => rel((await import("./chatParticipant")).chatParticipant, "list", { omit: "chat" }),
        invites: async () => rel((await import("./chatInvite")).chatInvite, "list", { omit: "chat" }),
        labels: { __use: 0 },
        // Messages are omitted here because they are handled by the chatMessageTree query
        translations: () => rel(chatTranslation, "full"),
    },
    list: {
        __define: {
            0: async () => rel((await import("./label")).label, "list"),
        },
        labels: { __use: 0 },
        translations: () => rel(chatTranslation, "list"),
    },
};
