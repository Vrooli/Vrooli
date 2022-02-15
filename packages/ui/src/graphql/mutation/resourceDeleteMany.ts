import { gql } from 'graphql-tag';

export const resourceDeleteManyMutation = gql`
    mutation resourceDeleteMany($input: DeleteManyInput!) {
        resourceDeleteMany(input: $input) {
            count
        }
    }
`