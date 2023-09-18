import { AwardModelLogic } from "../base/types";
import { Formatter } from "../types";

export const AwardFormat: Formatter<AwardModelLogic> = {
    gqlRelMap: {
        __typename: "Award",
    },
    prismaRelMap: {
        __typename: "Award",
        user: "User",
    },
    countFields: {},
};
