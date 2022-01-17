import { gql } from 'graphql-tag';

export const voteAddMutation = gql`
    mutation voteAdd($input: VoteInput!) {
        voteAdd(input: $input) {
            success
        }
    }
`