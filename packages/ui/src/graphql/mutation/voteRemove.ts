import { gql } from 'graphql-tag';

export const voteRemoveMutation = gql`
    mutation voteRemove($input: VoteRemoveInput!) {
        voteRemove(input: $input) {
            success
        }
    }
`