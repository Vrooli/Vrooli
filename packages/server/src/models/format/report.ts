import { ReportModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ReportFormat: Formatter<ReportModelLogic> = {
    gqlRelMap: {
        __typename: "Report",
        responses: "ReportResponse",
    },
    prismaRelMap: {
        __typename: "Report",
        responses: "ReportResponse",
    },
    hiddenFields: ["createdById"], // Always hide report creator,
    countFields: {
        responsesCount: true,
    },
};
