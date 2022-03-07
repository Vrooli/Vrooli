import { gql } from 'graphql-tag';

export const commentFields = gql`
    fragment commentFields on Comment {
        id
        created_at
        updated_at
        score
        isUpvoted
        role
        isStarred
        commentedOn {
            ... on Project {
                id
                translations {
                    id
                    language
                    name
                }
            }
            ... on Routine {
                id
                translations {
                    id
                    language
                    title
                }
            }
            ... on Standard {
                id
                name
            }
        }
        creator {
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
        translations {
            id
            language
            text
        }
    }
`