import { gql } from 'graphql-tag';
import { apiFields } from 'graphql/fragment';

export const apiQuery = gql`
    ${apiFields}
    query api($input: FindByIdInput!) {
        api(input: $input) {
            ...apiFields
        }
    }
`