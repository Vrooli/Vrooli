import { gql } from 'graphql-tag';
import { stepInputDataFields } from 'graphql/fragment';

export const stepInputDataUpdateMutation = gql`
    ${stepInputDataFields}
    mutation stepInputDataUpdate($input: StepInputDataUpdateInput!) {
        stepInputDataUpdate(input: $input) {
            ...stepInputDataFields
        }
    }
`