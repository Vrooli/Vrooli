import { rel } from "../utils";
export const chatInviteYou = {
    __typename: "ChatInviteYou",
    full: {
        canDelete: true,
        canUpdate: true,
    },
};
export const chatInvite = {
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
//# sourceMappingURL=chatInvite.js.map