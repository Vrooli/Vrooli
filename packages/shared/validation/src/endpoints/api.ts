import * as yup from 'yup';
import { id, rel, req, YupModel } from "../utils";
import { tagValidation } from './tag';
import { labelValidation } from './label';
import { apiVersionValidation } from './apiVersion';

export const apiValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        ...rel('user', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'many', 'opt'),
        ...rel('parent', ['Connect'], 'many', 'opt'),
        ...rel('tags', ['Connect', 'Create'], 'many', 'opt', tagValidation),
        ...rel('versions', ['Create'], 'many', 'opt', apiVersionValidation),
        ...rel('labels', ['Connect', 'Create'], 'many', 'opt', labelValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        ...rel('user', ['Connect'], 'one', 'opt'),
        ...rel('organization', ['Connect'], 'many', 'opt'),
        ...rel('tags', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', tagValidation),
        ...rel('versions', ['Create', 'Update', 'Delete'], 'many', 'opt', apiVersionValidation),
        ...rel('labels', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', labelValidation),
    }),
}