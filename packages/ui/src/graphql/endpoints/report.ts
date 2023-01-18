import { reportFields as fullFields, listReportFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reportEndpoint = {
    findOne: toQuery('report', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('reports', 'ReportSearchInput', toSearch(listFields)),
    create: toMutation('reportCreate', 'ReportCreateInput', fullFields[1]),
    update: toMutation('reportUpdate', 'ReportUpdateInput', fullFields[1])
}