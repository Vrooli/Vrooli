/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in reportResponse.test.ts
import { ReportSuggestedAction } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { details, id, language } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

function actionSuggested() {
    return enumToYup(ReportSuggestedAction);
}

export const reportResponseValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        actionSuggested: req(actionSuggested()),
        details: opt(details),
        language: opt(language),
    }, [
        ["report", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        actionSuggested: opt(actionSuggested()),
        details: opt(details),
        language: opt(language),
    }, [], [], d),
};
