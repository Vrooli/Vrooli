import { resourceFields } from "./resource";
import { tagFields } from "./tag";

export const organizationNameFields = ['Organization', `{
    id
    handle
    translatedName
}`] as const;
export const organizationTranslationFields = ['OrganizationTranslation', `{
    id
    language
    bio
    name
}`] as const;
export const organizationPermissionFields = ['OrganizationPermission', `{
    canAddMembers
    canDelete
    canEdit
    canStar
    canReport
    canView
    isMember
}`] as const;
export const listOrganizationFields = ['Organization', `{
    id
    commentsCount
    handle
    stars
    isOpenToNewMembers
    isPrivate
    isStarred
    membersCount
    reportsCount
    permissionsOrganization ${organizationPermissionFields[1]}
    tags ${tagFields[1]}
    translations ${organizationTranslationFields[1]}
}`] as const;
export const organizationFields = ['Organization', `{
    id
    created_at
    handle
    isOpenToNewMembers
    isPrivate
    isStarred
    stars
    reportsCount
    permissionsOrganization ${organizationPermissionFields[1]}
    resourceList ${resourceFields[1]}
    roles {
        id
        created_at
        updated_at
        title
        translations {
            id
            language
            description
        }
    }
    tags ${tagFields[1]}
    translations ${organizationTranslationFields[1]}
}`] as const;