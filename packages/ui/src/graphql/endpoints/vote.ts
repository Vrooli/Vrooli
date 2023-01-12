import { listVoteFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const voteEndpoint = {
    votes: toQuery('votes', 'VoteSearchInput', [listFields], toSearch(listFields)),
    vote: toMutation('vote', 'VoteInput', [], `success`),
}