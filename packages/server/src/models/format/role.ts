import { RoleModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RoleFormat: Formatter<RoleModelLogic> = {
    gqlRelMap: {
        __typename: "Role",
        members: "Member",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename: "Role",
        members: "Member",
        meetings: "Meeting",
        organization: "Organization",
    },
    countFields: {
        membersCount: true,
    },
};
