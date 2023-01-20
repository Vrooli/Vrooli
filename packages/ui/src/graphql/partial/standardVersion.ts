import { resourceListPartial } from "./resourceList";

export const listStandardVersionFields = ['StandardVersionVersion', `{
    id
}`] as const;
export const standardVersionFields = ['StandardVersionVersion', `{
    id
}`] as const;

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
    props
    reportsCount
    tags ${tagPartial.list}
    translations {
        id
        language
        name
        description
        jsonVariable
    }
    type
    you ${standardYouPartial.full}
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
    resourceList ${resourceListPartial.full}
    tags ${tagPartial.list}
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
    score
    you ${standardYouPartial.full
}`] as const;