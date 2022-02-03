import { gql } from 'graphql-tag';

export const commentDeleteOneMutation = gql`
    mutation commentDeleteOne($input: DeleteOneInput!) {
        commentDeleteOne(input: $input) {
            success
        }
    }
`