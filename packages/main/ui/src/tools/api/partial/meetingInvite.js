import { rel } from "../utils";
export const meetingInviteYou = {
    __typename: "MeetingInviteYou",
    full: {
        canDelete: true,
        canUpdate: true,
    },
};
export const meetingInvite = {
    __typename: "MeetingInvite",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        you: () => rel(meetingInviteYou, "full"),
    },
    full: {
        meeting: async () => rel((await import("./meeting")).meeting, "full", { omit: "invites" }),
    },
    list: {
        meeting: async () => rel((await import("./meeting")).meeting, "list", { omit: "invites" }),
    },
};
//# sourceMappingURL=meetingInvite.js.map