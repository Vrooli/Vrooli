import { gql } from 'graphql-tag';
import { resourceFields } from 'graphql/fragment';

export const resourceQuery = gql`
    ${resourceFields}
    query resource($input: FindByIdInput!) {
        resource(input: $input) {
            ...resourceFields
        }
    }
`