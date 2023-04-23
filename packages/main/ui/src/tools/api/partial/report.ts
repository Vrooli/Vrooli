import { Report, ReportYou } from "@local/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const reportYou: GqlPartial<ReportYou> = {
    __typename: "ReportYou",
    common: {
        canDelete: true,
        canRespond: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};

export const report: GqlPartial<Report> = {
    __typename: "Report",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        details: true,
        language: true,
        reason: true,
        responsesCount: true,
        you: () => rel(reportYou, "full"),
    },
    full: {
        responses: async () => rel((await import("./reportResponse")).reportResponse, "full", { omit: "report" }),
    },
    list: {},
};
