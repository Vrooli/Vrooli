import { pullRequestFields as fullFields, listPullRequestFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const pullRequestEndpoint = {
    findOne: toQuery('pullRequest', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('pullRequests', 'PullRequestSearchInput', toSearch(listFields)),
    create: toMutation('pullRequestCreate', 'PullRequestCreateInput', fullFields[1]),
    update: toMutation('pullRequestUpdate', 'PullRequestUpdateInput', fullFields[1]),
    accept: toMutation('pullRequestAcdept', 'FindByIdInput', fullFields[1]),
    reject: toMutation('pullRequestReject', 'FindByIdInput', fullFields[1]),
}