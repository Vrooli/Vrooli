import { issueFields as fullFields, listIssueFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const issueEndpoint = {
    findOne: toQuery('issue', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('issues', 'IssueSearchInput', toSearch(listFields)),
    create: toMutation('issueCreate', 'IssueCreateInput', fullFields[1]),
    update: toMutation('issueUpdate', 'IssueUpdateInput', fullFields[1]),
    close: toMutation('issueClose', 'IssueCloseInput', fullFields[1]),
}