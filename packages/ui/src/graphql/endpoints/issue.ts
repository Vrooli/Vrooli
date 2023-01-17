import { issueFields as fullFields, listIssueFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const issueEndpoint = {
    findOne: toQuery('issue', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('issues', 'IssueSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('issueCreate', 'IssueCreateInput', [fullFields], `...fullFields`),
    update: toMutation('issueUpdate', 'IssueUpdateInput', [fullFields], `...fullFields`),
    close: toMutation('issueClose', 'IssueCloseInput', [fullFields], `...fullFields`),
}