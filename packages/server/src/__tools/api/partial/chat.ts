import { Chat, ChatTranslation, ChatYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const chatTranslation: ApiPartial<ChatTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const chatYou: ApiPartial<ChatYou> = {
    common: {
        canDelete: true,
        canInvite: true,
        canUpdate: true,
    },
};

export const chat: ApiPartial<Chat> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        openToAnyoneWithInvite: true,
        participants: async () => rel((await import("./chatParticipant.js")).chatParticipant, "list", { omit: "chat" }),
        restrictedToRoles: async () => rel((await import("./role.js")).role, "full"),
        team: async () => rel((await import("./team.js")).team, "nav"),
        participantsCount: true,
        invitesCount: true,
        you: () => rel(chatYou, "full"),
    },
    full: {
        participants: async () => rel((await import("./chatParticipant.js")).chatParticipant, "list", { omit: "chat" }),
        invites: async () => rel((await import("./chatInvite.js")).chatInvite, "list", { omit: "chat" }),
        labels: async () => rel((await import("./label.js")).label, "full"),
        // Messages are omitted here because they are handled by the chatMessageTree query
        translations: () => rel(chatTranslation, "full"),
    },
    list: {
        labels: async () => rel((await import("./label.js")).label, "list"),
        translations: () => rel(chatTranslation, "list"),
    },
};
