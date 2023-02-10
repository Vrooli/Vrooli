import { description, enumToYup, id, name, opt, referencing, req, transRel, YupModel, yupObj } from "../utils";
import { QuestionForType } from "@shared/consts";

const forType = enumToYup(QuestionForType);

export const questionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const questionValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        referencing: opt(referencing),
        for: req(forType),
    }, [
        ['for', ['Connect'], 'one', 'opt'],
        ['translations', ['Create'], 'many', 'opt', questionTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ['acceptedAnswer', ['Connect'], 'one', 'opt'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', questionTranslationValidation],
    ], [], o),
}