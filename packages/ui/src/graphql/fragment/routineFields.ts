import { gql } from 'graphql-tag';

export const routineFields = gql`
    fragment routineTagFields on Tag {
        id
        tag
        translations {
            id
            language
            description
        }
    }
    fragment routineFields on Routine {
        id
        completedAt
        created_at
        isAutomatable
        isInternal
        isComplete
        isStarred
        isUpvoted
        role
        score
        stars
        tags {
            ...routineTagFields
        }
        translations {
            id
            language
            description
            title
        }
        version
    }
`