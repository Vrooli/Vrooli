import { details, id, language, opt, reportCreatedFor, reportReason, req, yupObj } from "../utils";
export const reportValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        createdFor: req(reportCreatedFor),
        details: opt(details),
        language: req(language),
        reason: req(reportReason),
    }, [
        ["createdFor", ["Connect"], "one", "req"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        details: opt(details),
        language: opt(language),
        reason: opt(reportReason),
    }, [], [], o),
};
//# sourceMappingURL=report.js.map