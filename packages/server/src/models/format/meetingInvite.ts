import { Formatter } from "../types";

const __typename = "MeetingInvite" as const;
export const MeetingInviteFormat: Formatter<ModelMeetingInviteLogic> = {
    gqlRelMap: {
        __typename,
        meeting: "Meeting",
        user: "User",
    },
    prismaRelMap: {
        __typename,
        meeting: "Meeting",
        user: "User",
    },
    countFields: {},
};
