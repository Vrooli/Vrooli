import { ReportResponseModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ReportResponse" as const;
export const ReportResponseFormat: Formatter<ReportResponseModelLogic> = {
    gqlRelMap: {
        __typename,
        report: "Report",
    },
    prismaRelMap: {
        __typename,
        report: "Report",
    },
    hiddenFields: ["createdById"], // Always hide report creator
    countFields: {
        responsesCount: true,
    },
};
