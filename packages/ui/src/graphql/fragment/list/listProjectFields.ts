import { gql } from 'graphql-tag';

export const listProjectFields = gql`
    fragment listProjectTagFields on Tag {
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
        commentsCount
        handle
        role
        score
        stars
        isUpvoted
        isStarred
        reportsCount
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