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
        commentsCount
        default
        score
        stars
        isDeleted
        isInternal
        isPrivate
        isUpvoted
        isStarred
        name
        props
        reportsCount
        permissionsStandard {
            canComment
            canDelete
            canEdit
            canStar
            canReport
            canVote
        }
        tags {
            ...listStandardTagFields
        }
        translations {
            id
            language
            description
            jsonVariable
        }
        type
        version
        versionGroupId
        yup
    }
`