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
        score
        stars
        isUpvoted
        isStarred
        name
        role
        tags {
            ...listStandardTagFields
        }
        translations {
            id
            language
            description
        }
    }
`