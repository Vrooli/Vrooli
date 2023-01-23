import { runProjectSchedulePartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const runProjectScheduleEndpoint = {
    findOne: toQuery('runProjectSchedule', 'FindByIdInput', runProjectSchedulePartial, 'full'),
    findMany: toQuery('runProjectSchedules', 'RunProjectScheduleSearchInput', ...toSearch(runProjectSchedulePartial)),
    create: toMutation('runProjectScheduleCreate', 'RunProjectScheduleCreateInput', runProjectSchedulePartial, 'full'),
    update: toMutation('runProjectScheduleUpdate', 'RunProjectScheduleUpdateInput', runProjectSchedulePartial, 'full')
}