import { Meeting, MeetingTranslation, MeetingYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const meetingTranslation: GqlPartial<MeetingTranslation> = {
    __typename: "MeetingTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        link: true,
        name: true,
    },
    full: {},
    list: {},
};

export const meetingYou: GqlPartial<MeetingYou> = {
    __typename: "MeetingYou",
    common: {
        canDelete: true,
        canInvite: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};

export const meeting: GqlPartial<Meeting> = {
    __typename: "Meeting",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        openToAnyoneWithInvite: true,
        showOnTeamProfile: true,
        team: async () => rel((await import("./team")).team, "nav"),
        restrictedToRoles: async () => rel((await import("./role")).role, "full"),
        attendeesCount: true,
        invitesCount: true,
        you: () => rel(meetingYou, "full"),
    },
    full: {
        __define: {
            0: async () => rel((await import("./label")).label, "full"),
        },
        attendees: async () => rel((await import("./user")).user, "nav"),
        invites: async () => rel((await import("./meetingInvite")).meetingInvite, "list", { omit: "meeting" }),
        labels: { __use: 0 },
        schedule: async () => rel((await import("./schedule")).schedule, "list", { omit: "meetings" }),
        translations: () => rel(meetingTranslation, "full"),
    },
    list: {
        __define: {
            0: async () => rel((await import("./label")).label, "list"),
        },
        labels: { __use: 0 },
        schedule: async () => rel((await import("./schedule")).schedule, "list", { omit: "meetings" }),
        translations: () => rel(meetingTranslation, "list"),
    },
};
