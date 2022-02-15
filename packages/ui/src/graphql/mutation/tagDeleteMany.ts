import { gql } from 'graphql-tag';

export const tagDeleteManyMutation = gql`
    mutation tagDeleteMany($input: DeleteManyInput!) {
        tagDeleteMany(input: $input) {
            count
        }
    }
`