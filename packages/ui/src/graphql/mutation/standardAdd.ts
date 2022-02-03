import { gql } from 'graphql-tag';
import { standardFields } from 'graphql/fragment';

export const standardAddMutation = gql`
    ${standardFields}
    mutation standardAdd($input: StandardAddInput!) {
        standardAdd(input: $input) {
            ...standardFields
        }
    }
`