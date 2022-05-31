import { gql } from 'graphql-tag';
import { stepInputDataFields } from 'graphql/fragment';

export const stepInputDataCreateMutation = gql`
    ${stepInputDataFields}
    mutation stepInputDataCreate($input: StepInputDataCreateInput!) {
        stepInputDataCreate(input: $input) {
            ...stepInputDataFields
        }
    }
`