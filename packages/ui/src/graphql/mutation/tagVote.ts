import { gql } from 'graphql-tag';

export const tagVoteMutation = gql`
    mutation tagVote($input: TagVoteInput!) {
        tagVote(input: $input) {
            success
        }
    }
`