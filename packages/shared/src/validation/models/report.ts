import { details, id, language, opt, reportCreatedFor, reportReason, req, YupModel, yupObj } from "../utils";

export const reportValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        createdForType: req(reportCreatedFor),
        details: opt(details),
        language: req(language),
        reason: req(reportReason),
    }, [
        ["createdFor", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        details: opt(details),
        language: opt(language),
        reason: opt(reportReason),
    }, [], [], d),
};
