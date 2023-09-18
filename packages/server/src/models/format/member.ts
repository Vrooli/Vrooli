import { MemberModelLogic } from "../base/types";
import { Formatter } from "../types";

export const MemberFormat: Formatter<MemberModelLogic> = {
    gqlRelMap: {
        __typename: "Member",
        organization: "Organization",
        roles: "Role",
        user: "User",
    },
    prismaRelMap: {
        __typename: "Member",
        organization: "Organization",
        roles: "Role",
        user: "User",
    },
    countFields: {},
};
