import { gql } from 'graphql-tag';
import { commentFields } from 'graphql/fragment';

export const commentAddMutation = gql`
    ${commentFields}
    mutation commentAdd($input: CommentAddInput!) {
        commentAdd(input: $input) {
            ...commentFields
        }
    }
`