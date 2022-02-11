import { gql } from 'graphql-tag';

export const commentFields = gql`
    fragment commentFields on Comment {
        id
        text
        created_at
        updated_at
        score
        isUpvoted
        role
        isStarred
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
        commentedOn {
            ... on Project {
                id
                name
            }
            ... on Routine {
                id
                title
            }
            ... on Standard {
                id
                name
            }
        }
    }
`