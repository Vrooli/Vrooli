import { gql } from 'graphql-tag';

export const standardsCountQuery = gql`
    query standardsCount($input: StandardCountInput!) {
        standardsCount(input: $input)
    }
`