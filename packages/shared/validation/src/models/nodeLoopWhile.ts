import { description, id, name, req, opt, nodeCondition, YupModel, transRel, yupObj } from '../utils';

export const nodeLoopWhileTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const nodeLoopWhileValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        condition: req(nodeCondition),
    }, [
        ['to', ['Connect'], 'one', 'opt'],
        ['translations', ['Create'], 'many', 'opt', nodeLoopWhileTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        condition: opt(nodeCondition),
    }, [
        ['to', ['Connect'], 'one', 'opt'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', nodeLoopWhileTranslationValidation],
    ], [], o),
}