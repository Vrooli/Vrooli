import { MeetingInvite, MeetingInviteYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const meetingInviteYou: ApiPartial<MeetingInviteYou> = {
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const meetingInvite: ApiPartial<MeetingInvite> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        you: () => rel(meetingInviteYou, "full"),
    },
    full: {
        meeting: async () => rel((await import("./meeting.js")).meeting, "full", { omit: "invites" }),
    },
    list: {
        meeting: async () => rel((await import("./meeting.js")).meeting, "list", { omit: "invites" }),
    },
};
