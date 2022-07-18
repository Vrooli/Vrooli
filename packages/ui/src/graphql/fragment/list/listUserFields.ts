import { gql } from 'graphql-tag';

export const listUserFields = gql`
    fragment listUserFields on User {
        id
        handle
        name
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