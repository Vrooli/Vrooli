import { gql } from 'graphql-tag';
import { standardFields } from 'graphql/fragment';

export const standardCreateMutation = gql`
    ${standardFields}
    mutation standardCreate($input: StandardCreateInput!) {
        standardCreate(input: $input) {
            ...standardFields
        }
    }
`