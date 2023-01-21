import { reportResponsePartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reportResponseEndpoint = {
    findOne: toQuery('reportResponse', 'FindByIdInput', reportResponsePartial, 'full'),
    findMany: toQuery('reportResponses', 'ReportResponseSearchInput', ...toSearch(reportResponsePartial)),
    create: toMutation('reportResponseCreate', 'ReportResponseCreateInput', reportResponsePartial, 'full'),
    update: toMutation('reportResponseUpdate', 'ReportResponseUpdateInput', reportResponsePartial, 'full')
}