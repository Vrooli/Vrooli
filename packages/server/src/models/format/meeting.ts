import { MeetingModelLogic } from "../base/types";
import { Formatter } from "../types";

export const MeetingFormat: Formatter<MeetingModelLogic> = {
    gqlRelMap: {
        __typename: "Meeting",
        attendees: "User",
        invites: "MeetingInvite",
        labels: "Label",
        organization: "Organization",
        restrictedToRoles: "Role",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "Meeting",
        organization: "Organization",
        restrictedToRoles: "Role",
        attendees: "User",
        invites: "MeetingInvite",
        labels: "Label",
        schedule: "Schedule",
    },
    joinMap: {
        labels: "label",
        restrictedToRoles: "role",
        attendees: "user",
    },
    countFields: {
        attendeesCount: true,
        invitesCount: true,
        labelsCount: true,
        translationsCount: true,
    },
};
