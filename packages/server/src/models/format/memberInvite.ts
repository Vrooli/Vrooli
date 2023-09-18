import { MemberInviteModelLogic } from "../base/types";
import { Formatter } from "../types";

export const MemberInviteFormat: Formatter<MemberInviteModelLogic> = {
    gqlRelMap: {
        __typename: "MemberInvite",
        organization: "Organization",
        user: "User",
    },
    prismaRelMap: {
        __typename: "MemberInvite",
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};
