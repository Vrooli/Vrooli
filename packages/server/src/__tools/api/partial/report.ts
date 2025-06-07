import { type Report, type ReportYou } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reportYou: ApiPartial<ReportYou> = {
    common: {
        canDelete: true,
        canRespond: true,
        canUpdate: true,
        isOwn: true,
    },
};

export const report: ApiPartial<Report> = {
    common: {
        id: true,
        publicId: true,
        createdAt: true,
        updatedAt: true,
        details: true,
        language: true,
        reason: true,
        responsesCount: true,
        status: true,
        you: () => rel(reportYou, "full"),
    },
    full: {
        responses: async () => rel((await import("./reportResponse.js")).reportResponse, "full", { omit: "report" }),
    },
};
