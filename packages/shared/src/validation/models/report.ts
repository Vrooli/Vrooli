import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { details, id, language, reportCreatedFor, reportReason } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

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
