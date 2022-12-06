import { gql } from 'graphql-tag';

export const listRunRoutineFields = gql`
    fragment listRunRoutineTagFields on Tag {
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
    fragment listRunRoutineRoutineFields on Routine {
        id
        completedAt
        complexity
        created_at
        isAutomatable
        isDeleted
        isInternal
        isPrivate
        isComplete
        isStarred
        isUpvoted
        score
        simplicity
        stars
        permissionsRoutine {
            canDelete
            canEdit
            canFork
            canStar
            canReport
            canRun
            canVote
        }
        tags {
            ...listRunRoutineTagFields
        }
        translations {
            id
            language
            description
            title
        }
    }
    fragment listRunRoutineFields on RunRoutine {
        id
        completedComplexity
        contextSwitches
        isPrivate
        timeStarted
        timeElapsed
        timeCompleted
        title
        status
        routine {
            ...listRunRoutineRoutineFields
        }
    }
`