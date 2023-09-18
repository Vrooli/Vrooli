import { StatsOrganizationModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsOrganizationFormat: Formatter<StatsOrganizationModelLogic> = {
    gqlRelMap: {
        __typename: "StatsOrganization",
    },
    prismaRelMap: {
        __typename: "StatsOrganization",
        organization: "Organization",
    },
    countFields: {},
};
