import { successPartial, votePartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const voteEndpoint = {
    votes: toQuery('votes', 'VoteSearchInput', ...toSearch(votePartial)),
    vote: toMutation('vote', 'VoteInput', successPartial, 'full'),
}