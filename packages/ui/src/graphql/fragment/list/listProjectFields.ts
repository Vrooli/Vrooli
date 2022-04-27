import { gql } from 'graphql-tag';

export const listProjectFields = gql`
    fragment listProjectTagFields on Tag {
        id
        created_at
        isStarred
        stars
        tag
        translations {
            id
            language
            description
        }
    }
    fragment listProjectFields on Project {
        id
        handle
        role
        score
        stars
        isUpvoted
        isStarred
        tags {
            ...listProjectTagFields
        }
        translations {
            id
            language
            name
            description
        }
    }
`