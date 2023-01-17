import { resourceListFields } from "./resourceList";
import { rootPermissionFields } from "./root";
import { tagFields } from "./tag";

export const standardNameFields = ['Standard', `{
    id
    translatedName
}`] as const;
export const listStandardFields = ['Standard', `{
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
    props
    reportsCount
    permissionsRoot ${rootPermissionFields[1]}
    tags ${tagFields[1]}
    translations {
        id
        language
        name
        description
        jsonVariable
    }
    type
    yup
}`] as const;
export const standardFields = ['Standard', `{
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
    resourceList ${resourceListFields[1]}
    tags ${tagFields[1]}
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
}`] as const;