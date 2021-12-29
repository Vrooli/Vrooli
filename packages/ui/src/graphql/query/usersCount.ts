import { gql } from 'graphql-tag';

export const usersCountQuery = gql`
    query usersCount {
        usersCount
    }
`