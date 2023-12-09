import * as yup from "yup";
import { id, maxStrErr, minStrErr, opt, req, transRel, YupModel, yupObj } from "../utils";

const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(32768, maxStrErr);

export const questionAnswerTranslationValidation: YupModel = transRel({
    create: () => ({
        text: req(text),
    }),
    update: () => ({
        text: opt(text),
    }),
});

export const questionAnswerValidation: YupModel = {
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
