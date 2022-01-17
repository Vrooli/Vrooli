import { gql } from 'graphql-tag';

export const voteMutation = gql`
    mutation vote($input: VoteInput!) {
        vote(input: $input) {
            success
        }
    }
`