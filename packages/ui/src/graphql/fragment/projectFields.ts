import { gql } from 'graphql-tag';

export const projectFields = gql`
    fragment projectResourceFields on Resource {
        id
        created_at
        index
        link
        updated_at
        usedFor
        translations {
            id
            language
            description
            title
        }
    }
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
        resources {
            ...projectResourceFields
        }
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