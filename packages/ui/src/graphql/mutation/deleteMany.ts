import { gql } from 'graphql-tag';

export const deleteManyMutation = gql`
    mutation deleteMany($input: DeleteManyInput!) {
        deleteMany(input: $input) {
            count
        }
    }
`