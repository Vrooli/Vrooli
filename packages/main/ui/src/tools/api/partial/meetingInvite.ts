import { MeetingInvite, MeetingInviteYou } from "@local/shared;";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const meetingInviteYou: GqlPartial<MeetingInviteYou> = {
    __typename: "MeetingInviteYou",
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const meetingInvite: GqlPartial<MeetingInvite> = {
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
