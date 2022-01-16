import { gql } from 'graphql-tag';

export const voteDeleteOneMutation = gql`
    mutation voteDeleteOne($input: DeleteOneInput!) {
        voteDeleteOne(input: $input) {
            success
        }
    }
`