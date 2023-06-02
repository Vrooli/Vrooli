import { StatsOrganizationModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StatsOrganization" as const;
export const StatsOrganizationFormat: Formatter<StatsOrganizationModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        organization: "Organization",
    },
    countFields: {},
};
