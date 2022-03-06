import { gql } from 'graphql-tag';

export const projectFields = gql`
    fragment projectTagFields on Tag {
        id
        description
        tag
    }
    fragment projectFields on Project {
        id
        completedAt
        created_at
        description
        isComplete
        isStarred
        isUpvoted
        name
        role
        score
        stars
        tags {
            ...projectTagFields
        }
        owner {
            ... on Organization {
                id
                name
            }
            ... on User {
                id
                username
            }
        }
    }
`