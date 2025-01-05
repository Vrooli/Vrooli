import { Report, ReportYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const reportYou: GqlPartial<ReportYou> = {
    common: {
        canDelete: true,
        canRespond: true,
        canUpdate: true,
        isOwn: true,
    },
};

export const report: GqlPartial<Report> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        details: true,
        language: true,
        reason: true,
        responsesCount: true,
        status: true,
        you: () => rel(reportYou, "full"),
    },
    full: {
        responses: async () => rel((await import("./reportResponse")).reportResponse, "full", { omit: "report" }),
    },
};
