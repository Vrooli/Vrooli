import { id, req, opt, rel, YupModel } from '../utils';
import * as yup from 'yup';
import { nodeRoutineListItemValidation } from './nodeRoutineListItem';

export const nodeRoutineListValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        isOrdered: opt(yup.boolean()),
        isOptional: opt(yup.boolean()),
        ...rel('items', ['Create'], 'many', 'opt', nodeRoutineListItemValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        isOrdered: opt(yup.boolean()),
        isOptional: opt(yup.boolean()),
        ...rel('items', ['Create', 'Update', 'Delete'], 'many', 'opt', nodeRoutineListItemValidation),
    })
}