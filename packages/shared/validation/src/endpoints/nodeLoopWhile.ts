import { description, id, name, language, req, opt, rel, nodeCondition, YupModel, transRel } from '../utils';
import * as yup from 'yup';

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
    create: yup.object().shape({
        id: req(id),
        condition: req(nodeCondition),
        ...rel('to', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', nodeLoopWhileTranslationValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        condition: opt(nodeCondition),
        ...rel('to', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', nodeLoopWhileTranslationValidation),
    })
}