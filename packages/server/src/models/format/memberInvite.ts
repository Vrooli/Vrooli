import { Formatter } from "../types";

const __typename = "MemberInvite" as const;
export const MemberInviteFormat: Formatter<ModelMemberInviteLogic> = {
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
