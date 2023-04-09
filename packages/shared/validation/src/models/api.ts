import { bool, id, opt, req, YupModel, yupObj } from "../utils";
import { tagValidation } from './tag';
import { labelValidation } from './label';
import { apiVersionValidation } from './apiVersion';

export const apiValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ['user', ['Connect'], 'one', 'opt'],
        ['organization', ['Connect'], 'one', 'opt'],
        ['parent', ['Connect'], 'one', 'opt'],
        ['tags', ['Connect', 'Create'], 'many', 'opt', tagValidation],
        ['versions', ['Create'], 'many', 'opt', apiVersionValidation, ['root']],
        ['labels', ['Connect', 'Create'], 'many', 'opt', labelValidation],
    ], [['organizationConnect', 'userConnect']], o),
    update: ({ o }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ['user', ['Connect'], 'one', 'opt'],
        ['organization', ['Connect'], 'one', 'opt'],
        ['tags', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', tagValidation],
        ['versions', ['Create', 'Update', 'Delete'], 'many', 'opt', apiVersionValidation, ['root']],
        ['labels', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', labelValidation],
    ], [['organizationConnect', 'userConnect']], o),
}