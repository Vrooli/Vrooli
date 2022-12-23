import { reportFields as fullFields, listReportFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reportEndpoint = {
    findOne: toQuery('report', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('reports', 'ReportSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('reportCreate', 'ReportCreateInput', [fullFields], `...fullFields`),
    update: toMutation('reportUpdate', 'ReportUpdateInput', [fullFields], `...fullFields`)
}