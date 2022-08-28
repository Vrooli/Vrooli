import { gql } from 'graphql-tag';

export const listProjectFields = gql`
    fragment listProjectTagFields on Tag {
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
    fragment listProjectFields on Project {
        id
        commentsCount
        handle
        score
        stars
        isComplete
        isPrivate
        isUpvoted
        isStarred
        reportsCount
        permissionsProject {
            canComment
            canDelete
            canEdit
            canStar
            canReport
            canVote
        }
        tags {
            ...listProjectTagFields
        }
        translations {
            id
            language
            name
            description
        }
    }
`