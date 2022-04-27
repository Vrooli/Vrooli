import { gql } from 'graphql-tag';

export const listRoutineFields = gql`
    fragment listRoutineTagFields on Tag {
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
    fragment listRoutineFields on Routine {
        id
        completedAt
        complexity
        created_at
        isAutomatable
        isInternal
        isComplete
        isStarred
        isUpvoted
        role
        score
        simplicity
        stars
        tags {
            ...listRoutineTagFields
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