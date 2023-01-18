import { reportResponseFields as fullFields, listReportResponseFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reportResponseEndpoint = {
    findOne: toQuery('reportResponse', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('reportResponses', 'ReportResponseSearchInput', toSearch(listFields)),
    create: toMutation('reportResponseCreate', 'ReportResponseCreateInput', fullFields[1]),
    update: toMutation('reportResponseUpdate', 'ReportResponseUpdateInput', fullFields[1])
}