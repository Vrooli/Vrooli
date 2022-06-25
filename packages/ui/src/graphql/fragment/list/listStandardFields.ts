import { gql } from 'graphql-tag';

export const listStandardFields = gql`
    fragment listStandardTagFields on Tag {
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
    fragment listStandardFields on Standard {
        id
        commentsCount
        score
        stars
        isUpvoted
        isStarred
        name
        reportsCount
        role
        tags {
            ...listStandardTagFields
        }
        translations {
            id
            language
            description
        }
        type
    }
`