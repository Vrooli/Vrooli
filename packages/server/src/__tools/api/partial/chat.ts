import { Chat, ChatTranslation, ChatYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const chatTranslation: GqlPartial<ChatTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const chatYou: GqlPartial<ChatYou> = {
    common: {
        canDelete: true,
        canInvite: true,
        canUpdate: true,
    },
};

export const chat: GqlPartial<Chat> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        openToAnyoneWithInvite: true,
        participants: async () => rel((await import("./chatParticipant")).chatParticipant, "list", { omit: "chat" }),
        restrictedToRoles: async () => rel((await import("./role")).role, "full"),
        team: async () => rel((await import("./team")).team, "nav"),
        participantsCount: true,
        invitesCount: true,
        you: () => rel(chatYou, "full"),
    },
    full: {
        participants: async () => rel((await import("./chatParticipant")).chatParticipant, "list", { omit: "chat" }),
        invites: async () => rel((await import("./chatInvite")).chatInvite, "list", { omit: "chat" }),
        labels: async () => rel((await import("./label")).label, "full"),
        // Messages are omitted here because they are handled by the chatMessageTree query
        translations: () => rel(chatTranslation, "full"),
    },
    list: {
        labels: async () => rel((await import("./label")).label, "list"),
        translations: () => rel(chatTranslation, "list"),
    },
};
