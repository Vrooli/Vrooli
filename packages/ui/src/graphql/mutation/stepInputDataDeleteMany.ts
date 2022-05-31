import { gql } from 'graphql-tag';

export const stepInputDataDeleteManyMutation = gql`
    mutation stepInputDataDeleteMany($input: DeleteManyInput!) {
        stepInputDataDeleteMany(input: $input) {
            count
        }
    }
`