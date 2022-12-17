import { blankToUndefined, id, rel, req, YupModel } from '../utils';
import * as yup from 'yup';
import { tagValidation } from './tag';

const userScheduleFilterType = yup.string().transform(blankToUndefined).oneOf([
    'Blur',
    'Hide',
    'ShowMore',
]);

export const userScheduleFilterValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        filterType: req(userScheduleFilterType),
        ...rel('userSchedule', ['Connect'], 'one', 'opt'),
        ...rel('tags', ['Create', 'Connect'], 'one', 'opt', tagValidation),
    }),
    update: yup.object().shape({ }), // Can only create and delete
}