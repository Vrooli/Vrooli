import { gql } from 'graphql-tag';
import { commentFields } from 'graphql/fragment';

export const commentQuery = gql`
    ${commentFields}
    query comment($input: FindByIdInput!) {
        comment(input: $input) {
            ...commentFields
        }
    }
`