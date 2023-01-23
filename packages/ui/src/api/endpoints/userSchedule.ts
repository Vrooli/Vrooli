import { userSchedulePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const userScheduleEndpoint = {
    findOne: toQuery('userSchedule', 'FindByIdInput', userSchedulePartial, 'full'),
    findMany: toQuery('userSchedules', 'UserScheduleSearchInput', ...toSearch(userSchedulePartial)),
    create: toMutation('userScheduleCreate', 'UserScheduleCreateInput', userSchedulePartial, 'full'),
    update: toMutation('userScheduleUpdate', 'UserScheduleUpdateInput', userSchedulePartial, 'full')
}