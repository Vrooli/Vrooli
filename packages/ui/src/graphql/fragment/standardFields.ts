import { gql } from 'graphql-tag';

export const standardFields = gql`
    fragment standardTagFields on Tag {
        id
        description
        tag
    }
    fragment standardFields on Standard {
        id
        name
        description
        role
        type
        schema
        default
        isFile
        created_at
        tags {
            ...standardTagFields
        }
        creator {
            ... on Organization {
                id
                name
            }
            ... on User {
                id
                username
            }
        }
        stars
        isStarred
        score
        isUpvoted
    }
`