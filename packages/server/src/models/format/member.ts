import { MemberModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Member" as const;
export const MemberFormat: Formatter<MemberModelLogic> = {
    gqlRelMap: {
        __typename,
        organization: "Organization",
        roles: "Role",
        user: "User",
    },
    prismaRelMap: {
        __typename,
        organization: "Organization",
        roles: "Role",
        user: "User",
    },
    countFields: {},
};
