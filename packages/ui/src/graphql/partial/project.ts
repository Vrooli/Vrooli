import { ProjectYou } from "@shared/consts";
import { GqlPartial } from "types";
import { resourceListPartial } from "./resourceList";
import { tagPartial } from "./tag";

export const projectYouPartial: GqlPartial<ProjectYou> = {
    __typename: 'ProjectYou',
    full: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canTransfer: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    },
}

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