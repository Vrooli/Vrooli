import { gql } from 'graphql-tag';
import { stepInputDataFields } from 'graphql/fragment';

export const stepInputDataQuery = gql`
    ${stepInputDataFields}
    query stepInputData($input: FindByIdInput!) {
        stepInputData(input: $input) {
            ...stepInputDataFields
        }
    }
`