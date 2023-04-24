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
        openToAnyoneWithInvite: true,
        showOnOrganizationProfile: true,
        organization: async () => rel((await import("./organization")).organization, "nav"),
        restrictedToRoles: async () => rel((await import("./role")).role, "full"),
        attendeesCount: true,
        invitesCount: true,
        you: () => rel(meetingYou, "full"),
    },
    full: {
        __define: {
            0: async () => rel((await import("./label")).label, "full"),
            1: async () => rel((await import("./schedule")).schedule, "full"),
        },
        attendees: async () => rel((await import("./user")).user, "nav"),
        invites: async () => rel((await import("./meetingInvite")).meetingInvite, "list", { omit: "meeting" }),
        labels: { __use: 0 },
        schedule: { __use: 1 },
        translations: () => rel(meetingTranslation, "full"),
    },
    list: {
        __define: {
            0: async () => rel((await import("./label")).label, "list"),
            1: async () => rel((await import("./schedule")).schedule, "list"),
        },
        labels: { __use: 0 },
        schedule: { __use: 1 },
        translations: () => rel(meetingTranslation, "list"),
    },
};
