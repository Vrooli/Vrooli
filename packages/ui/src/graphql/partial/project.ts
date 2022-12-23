import { resourceListFields } from "./resourceList";
import { rootPermissionFields } from "./root";
import { tagFields } from "./tag";

export const projectNameFields = ['Project', `{
    id
    handle
    translatedName
}`] as const;
export const listProjectFields = ['Project', `{
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
    permissionsRoot ${rootPermissionFields[1]}
    tags ${tagFields[1]}
    translations {
        id
        language
        name
        description
    }
}`] as const;
export const projectFields = ['Project', `{
    id
    completedAt
    created_at
    handle
    isComplete
    isPrivate
    isStarred
    isUpvoted
    score
    stars
    permissionsRoot ${rootPermissionFields[1]}
    resourceList ${resourceListFields[1]}
    tags ${tagFields[1]}
    translations {
        id
        language
        description
        name
    }
    owner {
        ... on Organization {
            id
            handle
            translations {
                id
                language
                name
            }
            permissionsOrganization {
                canAddMembers
                canDelete
                canEdit
                canStar
                canReport
                isMember
            }
        }
        ... on User {
            id
            name
            handle
        }
    }
}`] as const;