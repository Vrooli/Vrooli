import { reportPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const reportEndpoint = {
    findOne: toQuery('report', 'FindByIdInput', reportPartial, 'full'),
    findMany: toQuery('reports', 'ReportSearchInput', ...toSearch(reportPartial)),
    create: toMutation('reportCreate', 'ReportCreateInput', reportPartial, 'full'),
    update: toMutation('reportUpdate', 'ReportUpdateInput', reportPartial, 'full')
}