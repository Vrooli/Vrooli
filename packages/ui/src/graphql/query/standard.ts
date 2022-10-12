import { gql } from 'graphql-tag';
import { standardFields } from 'graphql/fragment';

export const standardQuery = gql`
    ${standardFields}
    query standard($input: FindByVersionInput!) {
        standard(input: $input) {
            ...standardFields
        }
    }
`