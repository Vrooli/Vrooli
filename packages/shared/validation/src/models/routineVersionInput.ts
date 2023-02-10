import { bool, description, id, index, instructions, name, opt, req, transRel, YupModel, yupObj } from "../utils";

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
    create: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        isRequired: opt(bool),
        name: opt(name),
    }, [
        ['routineVersion', ['Connect'], 'one', 'req'],
        ['standardVersion', ['Connect', 'Create'], 'one', 'req'],
        ['translations', ['Create'], 'many', 'opt', routineVersionInputTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        isRequired: opt(bool),
        name: opt(name),
    }, [
        ['standardVersion', ['Connect', 'Create', 'Disconnect'], 'one', 'req'],
        ['translations', ['Create'], 'many', 'opt', routineVersionInputTranslationValidation],
    ], [], o),
}