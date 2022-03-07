import { gql } from 'graphql-tag';

export const projectFields = gql`
    fragment projectTagFields on Tag {
        id
        tag
        translations {
            id
            language
            description
        }
    }
    fragment projectFields on Project {
        id
        completedAt
        created_at
        isComplete
        isStarred
        isUpvoted
        role
        score
        stars
        tags {
            ...projectTagFields
        }
        translations {
            id
            language
            description
            name
        }
        owner {
            ... on Organization {
                id
                translations {
                    id
                    language
                    name
                }
            }
            ... on User {
                id
                username
            }
        }
    }
`