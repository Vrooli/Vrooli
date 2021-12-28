import { gql } from 'graphql-tag';
import { standardFields } from 'graphql/fragment';

export const standardUpdateMutation = gql`
    ${standardFields}
    mutation standardUpdate($input: StandardInput!) {
        standardUpdate(input: $input) {
            ...standardFields
        }
    }
`