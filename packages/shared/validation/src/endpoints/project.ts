import { id, req, opt, YupModel, handle, rel, permissions } from '../utils';
import * as yup from 'yup';
import { tagValidation } from './tag';
import { projectVersionValidation } from './projectVersion';
import { labelValidation } from './label';

const isPrivate = yup.boolean()

export const projectValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        handle: opt(handle),
        isPrivate: opt(isPrivate),
        permissions: opt(permissions),
        ...rel('user', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'one', 'opt'),
        ...rel('parent', ['Connect'], 'one', 'opt'),
        ...rel('labels', ['Connect', 'Create'], 'many', 'opt', labelValidation),
        ...rel('versions', ['Create'], 'many', 'opt', projectVersionValidation),
        ...rel('tags', ['Connect', 'Create'], 'many', 'opt', tagValidation),
    }, [['userConnect', 'organizationConnect']]),
    update: () => yup.object().shape({
        id: req(id),
        handle: opt(handle),
        isPrivate: opt(isPrivate),
        permissions: opt(permissions),
        ...rel('user', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'one', 'opt'),
        ...rel('parent', ['Connect'], 'one', 'opt'),
        ...rel('labels', ['Connect', 'Create'], 'many', 'opt', labelValidation),
        ...rel('versions', ['Create'], 'many', 'opt', projectVersionValidation),
        ...rel('tags', ['Connect', 'Create'], 'many', 'opt', tagValidation),
    }, [['userConnect', 'organizationConnect']])
}