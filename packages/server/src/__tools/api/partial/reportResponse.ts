import { ReportResponse, ReportResponseYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reportResponseYou: ApiPartial<ReportResponseYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};


export const reportResponse: ApiPartial<ReportResponse> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        actionSuggested: true,
        details: true,
        language: true,
        report: async () => rel((await import("./report.js")).report, "nav", { omit: "responses" }),
        you: () => rel(reportResponseYou, "full"),
    },
};
