import { gql } from 'graphql-tag';
import { tagFields } from 'graphql/fragment';

export const tagQuery = gql`
    ${tagFields}
    query tag($input: FindByIdInput!) {
        tag(input: $input) {
            ...tagFields
        }
    }
`