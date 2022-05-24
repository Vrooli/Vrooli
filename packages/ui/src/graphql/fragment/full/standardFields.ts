import { gql } from 'graphql-tag';

export const standardFields = gql`
    fragment standardTagFields on Tag {
        id
        tag
        translations {
            id
            language
            description
        }
    }
    fragment standardFields on Standard {
        id
        name
        role
        type
        type
        props
        yup
        default
        created_at
        tags {
            ...standardTagFields
        }
        translations {
            id
            language
            description
        }
        creator {
            ... on Organization {
                id
                handle
                translations {
                    id
                    language
                    name
                }
            }
            ... on User {
                id
                name
                handle
            }
        }
        stars
        isStarred
        score
        isUpvoted
        version
    }
`