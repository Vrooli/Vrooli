import { gql } from 'graphql-tag';

export const userFields = gql`
    fragment userFields on User {
        id
        handle
        name
        created_at
        stars
        isStarred
        reportsCount
        translations {
            id
            language
            bio
        }
    }
`