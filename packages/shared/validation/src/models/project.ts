import { id, req, opt, YupModel, handle, permissions, yupObj, bool } from '../utils';
import { tagValidation } from './tag';
import { projectVersionValidation } from './projectVersion';
import { labelValidation } from './label';

export const projectValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        handle: opt(handle),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ['user', ['Connect'], 'one', 'opt'],
        ['organization', ['Connect'], 'one', 'opt'],
        ['parent', ['Connect'], 'one', 'opt'],
        ['labels', ['Connect', 'Create'], 'many', 'opt', labelValidation],
        ['versions', ['Create'], 'many', 'opt', projectVersionValidation, ['root']],
        ['tags', ['Connect', 'Create'], 'many', 'opt', tagValidation],
    ], [['organizationConnect', 'userConnect']], o),
    update: ({ o }) => yupObj({
        id: req(id),
        handle: opt(handle),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ['user', ['Connect'], 'one', 'opt'],
        ['organization', ['Connect'], 'one', 'opt'],
        ['parent', ['Connect'], 'one', 'opt'],
        ['labels', ['Connect', 'Create'], 'many', 'opt', labelValidation],
        ['versions', ['Create'], 'many', 'opt', projectVersionValidation, ['root']],
        ['tags', ['Connect', 'Create'], 'many', 'opt', tagValidation],
    ], [['organizationConnect', 'userConnect']], o),
}