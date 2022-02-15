import { gql } from 'graphql-tag';

export const projectFields = gql`
    fragment tagFields on Tag {
        id
        description
        tag
    }
    fragment projectFields on Project {
        id
        name
        description
        created_at
        tags {
            ...tagFields
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
        stars
        isStarred
        score
        role
        isUpvoted
    }
`