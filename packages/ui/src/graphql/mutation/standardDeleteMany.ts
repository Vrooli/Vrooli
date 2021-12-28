import { gql } from 'graphql-tag';

export const standardDeleteManyMutation = gql`
    mutation standardDeleteMany($input: DeleteManyInput!) {
        standardDeleteMany(input: $input) {
            count
        }
    }
`