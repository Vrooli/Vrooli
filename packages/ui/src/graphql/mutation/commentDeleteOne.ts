import { gql } from 'graphql-tag';

export const commentDeleteOneMutation = gql`
    mutation commentDeleteOne($input: DeleteCommentInput!) {
        commentDeleteOne(input: $input) {
            success
        }
    }
`