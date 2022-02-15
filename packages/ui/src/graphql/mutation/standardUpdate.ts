import { gql } from 'graphql-tag';
import { standardFields } from 'graphql/fragment';

export const standardUpdateMutation = gql`
    ${standardFields}
    mutation standardUpdate($input: StandardUpdateInput!) {
        standardUpdate(input: $input) {
            ...standardFields
        }
    }
`