import { ReportModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Report" as const;
export const ReportFormat: Formatter<ReportModelLogic> = {
    gqlRelMap: {
        __typename,
        responses: "ReportResponse",
    },
    prismaRelMap: {
        __typename,
        responses: "ReportResponse",
    },
    hiddenFields: ["createdById"], // Always hide report creator,
    countFields: {
        responsesCount: true,
    },
};
