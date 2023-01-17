import { description, id, name, opt, permissions, rel, req, transRel, YupModel } from '../utils';
import * as yup from 'yup';

export const roleTranslationValidation: YupModel = transRel({
    create: {
        description: req(description),
    },
    update: {
        description: opt(description),
    },
})

export const roleValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        name: req(name),
        permissions: opt(permissions),
        ...rel('members', ['Connect'], 'many', 'opt'),
        ...rel('organization', ['Connect'], 'one', 'req'),
        ...rel('translations', ['Create'], 'many', 'opt', roleTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        name: req(name),
        permissions: opt(permissions),
        ...rel('members', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', roleTranslationValidation),
    }),
}