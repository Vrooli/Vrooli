import { reportResponseFields as fullFields, listReportResponseFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reportResponseEndpoint = {
    findOne: toQuery('reportResponse', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('reportResponses', 'ReportResponseSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('reportResponseCreate', 'ReportResponseCreateInput', [fullFields], `...fullFields`),
    update: toMutation('reportResponseUpdate', 'ReportResponseUpdateInput', [fullFields], `...fullFields`)
}