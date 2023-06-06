import { MemberInviteModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "MemberInvite" as const;
export const MemberInviteFormat: Formatter<MemberInviteModelLogic> = {
    gqlRelMap: {
        __typename,
        organization: "Organization",
        user: "User",
    },
    prismaRelMap: {
        __typename,
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};
