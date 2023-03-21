import { description, id, name, opt, req, transRel, YupModel, yupObj } from '../utils';
import { resourceValidation } from './resource';

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
    create: ({ o }) => yupObj({
        id: req(id),
    }, [
        ['apiVersion', ['Connect'], 'one', 'opt'],
        ['focusMode', ['Connect'], 'one', 'opt'],
        ['organization', ['Connect'], 'one', 'opt'],
        ['post', ['Connect'], 'one', 'opt'],
        ['projectVersion', ['Connect'], 'one', 'opt'],
        ['routineVersion', ['Connect'], 'one', 'opt'],
        ['smartContractVersion', ['Connect'], 'one', 'opt'],
        ['translations', ['Create'], 'many', 'opt', resourceListTranslationValidation],
        ['resources', ['Create'], 'many', 'opt', resourceValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', resourceListTranslationValidation],
        ['resources', ['Create', 'Update', 'Delete'], 'many', 'opt', resourceValidation],
    ], [], o),
}