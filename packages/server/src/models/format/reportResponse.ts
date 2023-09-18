import { ReportResponseModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ReportResponseFormat: Formatter<ReportResponseModelLogic> = {
    gqlRelMap: {
        __typename: "ReportResponse",
        report: "Report",
    },
    prismaRelMap: {
        __typename: "ReportResponse",
        report: "Report",
    },
    hiddenFields: ["createdById"], // Always hide report creator
    countFields: {
        responsesCount: true,
    },
};
