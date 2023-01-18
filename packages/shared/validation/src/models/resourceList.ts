import { description, id, name, opt, rel, req, transRel, YupModel } from '../utils';
import { resourceValidation } from './resource';
import * as yup from 'yup';

export const resourceListTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: opt(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
})

export const resourceListValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        ...rel('apiVersion', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'one', 'opt'),
        ...rel('post', ['Connect'], 'one', 'opt'),
        ...rel('projectVersion', ['Connect'], 'one', 'opt'),
        ...rel('routineVersion', ['Connect'], 'one', 'opt'),
        ...rel('smartContractVersion', ['Connect'], 'one', 'opt'),
        ...rel('userSchedule', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', resourceListTranslationValidation),
        ...rel('resources', ['Create'], 'many', 'opt', resourceValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', resourceListTranslationValidation),
        ...rel('resources', ['Create', 'Update', 'Delete'], 'many', 'opt', resourceValidation),
    })
}