import { gql } from 'graphql-tag';

export const routineFields = gql`
    fragment routineTagFields on Tag {
        id
        description
        tag
    }
    fragment routineFields on Routine {
        id
        completedAt
        created_at
        description
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
        title
        version
    }
`