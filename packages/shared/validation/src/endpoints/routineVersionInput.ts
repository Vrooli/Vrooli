import * as yup from 'yup';
import { description, id, index, instructions, name, opt, rel, req, transRel, YupModel } from "../utils";

export const routineVersionInputTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        helpText: opt(instructions),
    },
    update: {
        description: opt(description),
        helpText: opt(instructions),
    }
})

export const routineVersionInputValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        index: opt(index),
        isRequired: opt(yup.boolean()),
        name: opt(name),
        ...rel('routineVersion', ['Connect'], 'one', 'req'),
        ...rel('standardVersion', ['Connect', 'Create'], 'one', 'req'),
        ...rel('translations', ['Create'], 'many', 'opt', routineVersionInputTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        index: opt(index),
        isRequired: opt(yup.boolean()),
        name: opt(name),
        ...rel('standardVersion', ['Connect', 'Create', 'Disconnect'], 'one', 'req'),
        ...rel('translations', ['Create'], 'many', 'opt', routineVersionInputTranslationValidation),
    }),
}