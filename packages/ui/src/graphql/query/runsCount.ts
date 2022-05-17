import { gql } from 'graphql-tag';

export const runsCountQuery = gql`
    query runsCount($input: RunCountInput!) {
        runsCount(input: $input)
    }
`