import { ProjectVersionTranslation, ProjectVersionYou } from "@shared/consts";
import { GqlPartial } from "types";

export const projectTranslationPartial: GqlPartial<ProjectVersionTranslation> = {
    __typename: 'ProjectVersionTranslation',
    full: () => ({
        id: true,
        language: true,
        description: true,
        name: true,
    }),
}

export const projectVersionYouPartial: GqlPartial<ProjectVersionYou> = {
    __typename: 'ProjectVersionYou',
    common: () => ({
        canComment: true,
        canCopy: true,
        canDelete: true,
        canEdit: true,
        canReport: true,
        canUse: true,
        canView: true,
    }),
    full: () => ({
        runs: runProjectPartial.full,
    }),
    list: () => ({
        runs: runProjectPartial.list,
    })
}

export const listProjectVersionFields = ['ProjectVersion', `{
    id
}`] as const;
export const projectVersionFields = ['ProjectVersion', `{
    id
}`] as const;

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
    reportsCount
    tags ${tagPartial.list}
    translations {
        id
        language
        name
        description
    }
    you ${projectYouPartial.full}
}`] as const;
export const projectFields = ['Project', `{
    id
    completedAt
    created_at
    handle
    isComplete
    isPrivate
    score
    stars
    resourceList ${resourceListPartial.full}
    tags ${tagPartial.list}
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
    you ${projectYouPartial.full}
}`] as const;