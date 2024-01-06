import { ReportSuggestedAction } from "@local/shared";
import { details, enumToYup, id, language, opt, req, YupModel, yupObj } from "../utils";

const actionSuggested = enumToYup(ReportSuggestedAction);

export const reportResponseValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        actionSuggested: req(actionSuggested),
        details: opt(details),
        language: opt(language),
    }, [
        ["report", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        actionSuggested: opt(actionSuggested),
        details: opt(details),
        language: opt(language),
    }, [], [], d),
};
