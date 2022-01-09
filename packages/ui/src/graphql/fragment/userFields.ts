import { gql } from 'graphql-tag';

export const userFields = gql`
    fragment userFields on User {
        id
        username
        stars
    }
`