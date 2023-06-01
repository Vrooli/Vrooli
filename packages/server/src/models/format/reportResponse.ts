import { Formatter } from "../types";

const __typename = "ReportResponse" as const;
export const ReportResponseFormat: Formatter<ModelReportResponseLogic> = {
    gqlRelMap: {
        __typename,
        report: "Report",
    },
    prismaRelMap: {
        __typename,
        report: "Report",
    },
    hiddenFields: ["createdById"], // Always hide report creator
};
