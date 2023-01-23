import { reportPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const reportEndpoint = {
    findOne: toQuery('report', 'FindByIdInput', reportPartial, 'full'),
    findMany: toQuery('reports', 'ReportSearchInput', ...toSearch(reportPartial)),
    create: toMutation('reportCreate', 'ReportCreateInput', reportPartial, 'full'),
    update: toMutation('reportUpdate', 'ReportUpdateInput', reportPartial, 'full')
}