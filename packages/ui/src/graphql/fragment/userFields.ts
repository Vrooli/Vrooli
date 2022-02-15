import { gql } from 'graphql-tag';

export const userFields = gql`
    fragment userFields on User {
        id
        username
        created_at
        stars
        isStarred
        bio
    }
`