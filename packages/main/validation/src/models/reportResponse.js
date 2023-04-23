import { ReportSuggestedAction } from "@local/consts";
import { details, enumToYup, id, language, opt, req, yupObj } from "../utils";
const actionSuggested = enumToYup(ReportSuggestedAction);
export const reportResponseValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        actionSuggested: req(actionSuggested),
        details: opt(details),
        language: opt(language),
    }, [
        ["report", ["Connect"], "one", "req"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        actionSuggested: opt(actionSuggested),
        details: opt(details),
        language: opt(language),
    }, [], [], o),
};
//# sourceMappingURL=reportResponse.js.map