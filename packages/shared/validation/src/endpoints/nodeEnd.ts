import { id, req, opt, rel, YupModel } from '../utils';
import * as yup from 'yup';

const wasSuccessful = yup.boolean()

export const nodeEndValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        wasSuccessful: opt(wasSuccessful),
        ...rel('node', ['Connect'], 'one', 'req'),
        ...rel('suggestedNextRoutineVersion', ['Connect'], 'many', 'opt'),
    }),
    update: () => yup.object().shape({
        id: req(id),
        wasSuccessful: opt(wasSuccessful),
        ...rel('suggestedNextRoutineVersion', ['Connect', 'Disconnect'], 'many', 'opt'),
    }),
}