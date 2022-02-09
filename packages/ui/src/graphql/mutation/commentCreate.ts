import { gql } from 'graphql-tag';
import { commentFields } from 'graphql/fragment';

export const commentCreateMutation = gql`
    ${commentFields}
    mutation commentCreate($input: CommentCreateInput!) {
        commentCreate(input: $input) {
            ...commentFields
        }
    }
`