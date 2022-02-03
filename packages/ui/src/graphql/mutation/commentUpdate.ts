import { gql } from 'graphql-tag';
import { commentFields } from 'graphql/fragment';

export const commentUpdateMutation = gql`
    ${commentFields}
    mutation commentUpdate($input: CommentUpdateInput!) {
        commentUpdate(input: $input) {
            ...commentFields
        }
    }
`