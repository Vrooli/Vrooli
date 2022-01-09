import { gql } from 'graphql-tag';

export const usersCountQuery = gql`
    query usersCount($input: UserCountInput!) {
        usersCount(input: $input)
    }
`