import { enumToYup, id, req, YupModel, yupObj } from '../utils';
import { tagValidation } from './tag';
import { UserScheduleFilterType } from '@shared/consts';

const userScheduleFilterType = enumToYup(UserScheduleFilterType);

export const userScheduleFilterValidation: YupModel<true, false> = {
    create: ({ o }) => yupObj({
        id: req(id),
        filterType: req(userScheduleFilterType),
    }, [
        ['userSchedule', ['Connect'], 'one', 'req'],
        ['tag', ['Create', 'Connect'], 'one', 'opt', tagValidation],
    ], [], o),
    // Can only create and delete
}