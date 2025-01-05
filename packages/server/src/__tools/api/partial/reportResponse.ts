import { ReportResponse, ReportResponseYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const reportResponseYou: GqlPartial<ReportResponseYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};


export const reportResponse: GqlPartial<ReportResponse> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        actionSuggested: true,
        details: true,
        language: true,
        report: async () => rel((await import("./report")).report, "nav", { omit: "responses" }),
        you: () => rel(reportResponseYou, "full"),
    },
};
