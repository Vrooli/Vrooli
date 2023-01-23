import { successPartial, votePartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const voteEndpoint = {
    votes: toQuery('votes', 'VoteSearchInput', ...toSearch(votePartial)),
    vote: toMutation('vote', 'VoteInput', successPartial, 'full'),
}