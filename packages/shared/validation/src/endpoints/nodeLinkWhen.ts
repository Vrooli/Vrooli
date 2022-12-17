import { description, id, name, req, opt, rel, nodeCondition, YupModel, transRel } from '../utils';
import * as yup from 'yup';

export const nodeLinkWhenTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
})

export const nodeLinkWhenValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        condition: req(nodeCondition),
        ...rel('link', ['Connect'], 'one', 'req'),
        ...rel('translations', ['Create'], 'many', 'opt', nodeLinkWhenTranslationValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        condition: opt(nodeCondition),
        ...rel('link', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Delete', 'Create', 'Update'], 'many', 'opt', nodeLinkWhenTranslationValidation),
    })
}