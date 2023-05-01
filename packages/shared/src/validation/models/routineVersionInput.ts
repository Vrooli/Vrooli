import { bool, description, id, index, instructions, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { standardVersionValidation } from "./standardVersion";

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
        ['standardVersion', ['Connect', 'Create'], 'one', 'req', standardVersionValidation],
        ['translations', ['Create'], 'many', 'opt', routineVersionInputTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        isRequired: opt(bool),
        name: opt(name),
    }, [
        ['standardVersion', ['Connect', 'Create', 'Disconnect'], 'one', 'req', standardVersionValidation],
        ['translations', ['Create'], 'many', 'opt', routineVersionInputTranslationValidation],
    ], [], o),
}