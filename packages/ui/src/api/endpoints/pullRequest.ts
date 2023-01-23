import { pullRequestPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const pullRequestEndpoint = {
    findOne: toQuery('pullRequest', 'FindByIdInput', pullRequestPartial, 'full'),
    findMany: toQuery('pullRequests', 'PullRequestSearchInput', ...toSearch(pullRequestPartial)),
    create: toMutation('pullRequestCreate', 'PullRequestCreateInput', pullRequestPartial, 'full'),
    update: toMutation('pullRequestUpdate', 'PullRequestUpdateInput', pullRequestPartial, 'full'),
    accept: toMutation('pullRequestAcdept', 'FindByIdInput', pullRequestPartial, 'full'),
    reject: toMutation('pullRequestReject', 'FindByIdInput', pullRequestPartial, 'full'),
}