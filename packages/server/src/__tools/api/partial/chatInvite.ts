import { ChatInvite, ChatInviteYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        user: async () => rel((await import("./user")).user, "nav"),
        you: () => rel(chatInviteYou, "full"),
    },
    full: {
        chat: async () => rel((await import("./chat")).chat, "full", { omit: "invites" }),
    },
    list: {
        chat: async () => rel((await import("./chat")).chat, "list", { omit: "invites" }),
    },
};
