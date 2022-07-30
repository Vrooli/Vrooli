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
            canStar
            canReport
            canRun
            canVote
        }
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
        versionGroupId
    }
    fragment listRunFields on Run {
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
            ...listRunRoutineFields
        }
    }
`