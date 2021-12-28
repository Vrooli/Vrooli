import { gql } from 'graphql-tag';

export const standardsCountQuery = gql`
    query standardsCount {
        standardsCount {
            count
        }
    }
`