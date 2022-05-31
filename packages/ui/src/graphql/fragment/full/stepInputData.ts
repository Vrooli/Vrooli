import { gql } from 'graphql-tag';

export const stepInputDataFields = gql`
    fragment stepInputDataFields on StepInputData {
        id
        stepId
        runId
        nodeId
        routineId
        subroutineId
        inputs {
            id
            inputId
            standardId
            name
            value
        }
    }
`