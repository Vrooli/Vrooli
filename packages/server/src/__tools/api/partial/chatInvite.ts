import { ChatInvite, ChatInviteYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const chatInviteYou: ApiPartial<ChatInviteYou> = {
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const chatInvite: ApiPartial<ChatInvite> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        user: async () => rel((await import("./user.js")).user, "nav"),
        you: () => rel(chatInviteYou, "full"),
    },
    full: {
        chat: async () => rel((await import("./chat.js")).chat, "full", { omit: "invites" }),
    },
    list: {
        chat: async () => rel((await import("./chat.js")).chat, "list", { omit: "invites" }),
    },
};
