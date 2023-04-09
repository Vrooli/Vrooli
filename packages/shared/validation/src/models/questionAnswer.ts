import * as yup from "yup";
import { blankToUndefined, id, maxStrErr, minStrErr, opt, req, transRel, YupModel, yupObj } from "../utils";

const text = yup.string().transform(blankToUndefined).min(1, minStrErr).max(32768, maxStrErr)

export const questionAnswerTranslationValidation: YupModel = transRel({
    create: {
        text: req(text),
    },
    update: {
        text: opt(text),
    }
})

export const questionAnswerValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
    }, [
        ['translations', ['Create'], 'many', 'opt', questionAnswerTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', questionAnswerTranslationValidation],
    ], [], o),
}