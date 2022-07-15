import { gql } from 'graphql-tag';

export const listRunFields = gql`
    fragment listRunTagFields on Tag {
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
    fragment listRunRoutineFields on Routine {
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
            ...listRunTagFields
        }
        translations {
            id
            language
            description
            title
        }
        version
    }
    fragment listRunFields on Run {
        id
        completedComplexity
        contextSwitches
        timeStarted
        timeElapsed
        timeCompleted
        title
        status
        routine {
            ...listRunRoutineFields
        }
    }
`