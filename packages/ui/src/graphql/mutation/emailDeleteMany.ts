import { gql } from 'graphql-tag';

export const emailDeleteManyMutation = gql`
    mutation emailDeleteMany($input: DeleteManyInput!) {
        emailDeleteMany(input: $input) {
            count
        }
    }
`