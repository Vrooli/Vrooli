import { successPartial, votePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const voteEndpoint = {
    votes: toQuery('votes', 'VoteSearchInput', ...toSearch(votePartial)),
    vote: toMutation('vote', 'VoteInput', successPartial, 'full'),
}