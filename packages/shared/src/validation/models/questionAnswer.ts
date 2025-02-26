import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { maxStrErr, minStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(32768, maxStrErr);

export const questionAnswerTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        text: req(text),
    }),
    update: () => ({
        text: opt(text),
    }),
});

export const questionAnswerValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Create"], "many", "opt", questionAnswerTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", questionAnswerTranslationValidation],
    ], [], d),
};
