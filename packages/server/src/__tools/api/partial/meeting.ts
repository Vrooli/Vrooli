import { Meeting, MeetingTranslation, MeetingYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const meetingTranslation: ApiPartial<MeetingTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        link: true,
        name: true,
    },
};

export const meetingYou: ApiPartial<MeetingYou> = {
    common: {
        canDelete: true,
        canInvite: true,
        canUpdate: true,
    },
};

export const meeting: ApiPartial<Meeting> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        openToAnyoneWithInvite: true,
        showOnTeamProfile: true,
        team: async () => rel((await import("./team.js")).team, "nav"),
        restrictedToRoles: async () => rel((await import("./role.js")).role, "full"),
        attendeesCount: true,
        invitesCount: true,
        you: () => rel(meetingYou, "full"),
    },
    full: {
        attendees: async () => rel((await import("./user.js")).user, "nav"),
        invites: async () => rel((await import("./meetingInvite.js")).meetingInvite, "list", { omit: "meeting" }),
        labels: async () => rel((await import("./label.js")).label, "full"),
        schedule: async () => rel((await import("./schedule.js")).schedule, "list", { omit: "meetings" }),
        translations: () => rel(meetingTranslation, "full"),
    },
    list: {
        labels: async () => rel((await import("./label.js")).label, "list"),
        schedule: async () => rel((await import("./schedule.js")).schedule, "list", { omit: "meetings" }),
        translations: () => rel(meetingTranslation, "list"),
    },
};
