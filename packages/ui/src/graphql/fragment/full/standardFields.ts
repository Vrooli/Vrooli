import { gql } from 'graphql-tag';

export const standardFields = gql`
    fragment standardResourceListFields on ResourceList {
        id
        created_at
        index
        usedFor
        translations {
            id
            language
            description
            title
        }
        resources {
            id
            created_at
            index
            link
            updated_at
            usedFor
            translations {
                id
                language
                description
                title
            }
        }
    }
    fragment standardTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment standardFields on Standard {
        id
        isDeleted
        isInternal
        isPrivate
        name
        type
        type
        props
        yup
        default
        created_at
        permissionsStandard {
            canComment
            canDelete
            canEdit
            canStar
            canReport
            canVote
        }
        resourceLists {
            ...standardResourceListFields
        }
        tags {
            ...standardTagFields
        }
        translations {
            id
            language
            description
            jsonVariable
        }
        creator {
            ... on Organization {
                id
                handle
                translations {
                    id
                    language
                    name
                }
            }
            ... on User {
                id
                name
                handle
            }
        }
        stars
        isStarred
        score
        isUpvoted
        version
        versionGroupId
    }
`