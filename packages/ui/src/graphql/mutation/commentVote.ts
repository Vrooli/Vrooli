import { gql } from 'graphql-tag';

export const commentVoteMutation = gql`
    mutation commentVote($input: VoteInput!) {
        commentVote(input: $input) {
            success
        }
    }
`