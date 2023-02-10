import { description, id, opt, req, transRel, YupModel, yupObj } from "../utils";

export const questionAnswerTranslationValidation: YupModel = transRel({
    create: {
        description: req(description),
    },
    update: {
        description: opt(description),
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