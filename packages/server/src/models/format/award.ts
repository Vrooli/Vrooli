import { AwardModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Award" as const;
export const AwardFormat: Formatter<AwardModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        user: "User",
    },
    countFields: {},
};
