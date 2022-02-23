import { gql } from 'graphql-tag';

export const routineFields = gql`
    fragment routineTagFields on Tag {
        id
        description
        tag
    }
    fragment routineFields on Routine {
        id
        version
        title
        description
        created_at
        isAutomatable
        role
        tags {
            ...routineTagFields
        }
        stars
        isStarred
        score
        isUpvoted
    }
`