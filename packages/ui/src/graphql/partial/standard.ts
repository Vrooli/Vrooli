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
}`] as const;