import { pullRequestFields as fullFields, listPullRequestFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const pullRequestEndpoint = {
    findOne: toQuery('pullRequest', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('pullRequests', 'PullRequestSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('pullRequestCreate', 'PullRequestCreateInput', [fullFields], `...fullFields`),
    update: toMutation('pullRequestUpdate', 'PullRequestUpdateInput', [fullFields], `...fullFields`),
    accept: toMutation('pullRequestAcdept', 'FindByIdInput', [fullFields], `...fullFields`),
    reject: toMutation('pullRequestReject', 'FindByIdInput', [fullFields], `...fullFields`),
}