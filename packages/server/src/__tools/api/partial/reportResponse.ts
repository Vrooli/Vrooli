import { type ReportResponse, type ReportResponseYou } from "@local/shared";
import { type ApiPartial } from "../types.js";
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
        createdAt: true,
        updatedAt: true,
        actionSuggested: true,
        details: true,
        language: true,
        report: async () => rel((await import("./report.js")).report, "nav", { omit: "responses" }),
        you: () => rel(reportResponseYou, "full"),
    },
};
