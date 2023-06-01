import { Formatter } from "../types";

const __typename = "Award" as const;
export const AwardFormat: Formatter<ModelAwardLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        user: "User",
    },
    countFields: {},
};
