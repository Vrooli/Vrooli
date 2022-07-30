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
        commentsCount
        completedAt
        complexity
        created_at
        isAutomatable
        isDeleted
        isInternal
        isComplete
        isPrivate
        isStarred
        isUpvoted
        reportsCount
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
            ...listRoutineTagFields
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
`