import { id, req, opt, rel, nodeOperation, YupModel } from '../utils';
import * as yup from 'yup';
import { nodeLinkWhenValidation } from './nodeLinkWhen';

export const nodeLinkValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        operation: opt(nodeOperation),
        ...rel('whens', ['Create'], 'many', 'opt', nodeLinkWhenValidation),
        ...rel('from', ['Connect'], 'one', 'req'),
        ...rel('to', ['Connect'], 'one', 'req'),
    }),
    update: () => yup.object().shape({
        id: req(id),
        operation: opt(nodeOperation),
        ...rel('whens', ['Delete', 'Create', 'Update'], 'many', 'opt', nodeLinkWhenValidation),
        ...rel('from', ['Connect'], 'one', 'opt'),
        ...rel('to', ['Connect'], 'one', 'opt'),
    })
}