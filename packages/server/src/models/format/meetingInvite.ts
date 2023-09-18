import { MeetingInviteModelLogic } from "../base/types";
import { Formatter } from "../types";

export const MeetingInviteFormat: Formatter<MeetingInviteModelLogic> = {
    gqlRelMap: {
        __typename: "MeetingInvite",
        meeting: "Meeting",
        user: "User",
    },
    prismaRelMap: {
        __typename: "MeetingInvite",
        meeting: "Meeting",
        user: "User",
    },
    countFields: {},
};
