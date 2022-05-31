import { gql } from 'graphql-tag';
import { stepInputDataFields } from 'graphql/fragment';

export const stepInputDatasQuery = gql`
    ${stepInputDataFields}
    query stepInputDatas($input: StepInputDataSearchInput!) {
        stepInputDatas(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...stepInputDataFields
                }
            }
        }
    }
`