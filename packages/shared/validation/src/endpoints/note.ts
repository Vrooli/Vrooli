import * as yup from 'yup';
import { id, opt, rel, req, YupModel } from "utils";
import { tagValidation } from './tag';
import { labelValidation } from './label';
import { noteVersionValidation } from './noteVersion';

export const noteValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        isPrivate: opt(yup.boolean()),
        ...rel('user', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'many', 'opt'),
        ...rel('parent', ['Connect'], 'many', 'opt'),
        ...rel('tags', ['Connect', 'Create'], 'many', 'opt', tagValidation),
        ...rel('versions', ['Create'], 'many', 'opt', noteVersionValidation),
        ...rel('labels', ['Connect', 'Create'], 'many', 'opt', labelValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        isPrivate: opt(yup.boolean()),
        ...rel('user', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'many', 'opt'),
        ...rel('tags', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', tagValidation),
        ...rel('versions', ['Create', 'Update', 'Delete'], 'many', 'opt', noteVersionValidation),
        ...rel('labels', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', labelValidation),
    }),
}