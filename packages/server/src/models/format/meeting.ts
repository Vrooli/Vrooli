import { MeetingModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Meeting" as const;
export const MeetingFormat: Formatter<MeetingModelLogic> = {
    gqlRelMap: {
        __typename,
        attendees: "User",
        invites: "MeetingInvite",
        labels: "Label",
        organization: "Organization",
        restrictedToRoles: "Role",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename,
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
        invites: "user",
    },
    countFields: {
        attendeesCount: true,
        invitesCount: true,
        labelsCount: true,
        translationsCount: true,
    },
};
