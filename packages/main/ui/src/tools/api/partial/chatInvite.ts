import { ChatInvite, ChatInviteYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const chatInviteYou: GqlPartial<ChatInviteYou> = {
    __typename: "ChatInviteYou",
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const chatInvite: GqlPartial<ChatInvite> = {
    __typename: "ChatInvite",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        you: () => rel(chatInviteYou, "full"),
    },
    full: {
        chat: async () => rel((await import("./chat")).chat, "full", { omit: "invites" }),
    },
    list: {
        chat: async () => rel((await import("./chat")).chat, "list", { omit: "invites" }),
    },
};
