import { issuePartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const issueEndpoint = {
    findOne: toQuery('issue', 'FindByIdInput', issuePartial, 'full'),
    findMany: toQuery('issues', 'IssueSearchInput', ...toSearch(issuePartial)),
    create: toMutation('issueCreate', 'IssueCreateInput', issuePartial, 'full'),
    update: toMutation('issueUpdate', 'IssueUpdateInput', issuePartial, 'full'),
    close: toMutation('issueClose', 'IssueCloseInput', issuePartial, 'full'),
}