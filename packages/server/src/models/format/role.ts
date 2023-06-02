import { RoleModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Role" as const;
export const RoleFormat: Formatter<RoleModelLogic> = {
    gqlRelMap: {
        __typename,
        members: "Member",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename,
        members: "Member",
        meetings: "Meeting",
        organization: "Organization",
    },
    countFields: {
        membersCount: true,
    },
};
