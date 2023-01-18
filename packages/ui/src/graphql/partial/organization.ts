import { Organization, OrganizationTranslation, OrganizationYou } from "@shared/consts";
import { GqlPartial } from "types";

export const organizationTranslationPartial: GqlPartial<OrganizationTranslation> = {
    __typename: 'OrganizationTranslation',
    full: {
        id: true,
        language: true,
        bio: true,
        name: true,
    },
}

export const organizationYouPartial: GqlPartial<OrganizationYou> = {
    __typename: 'OrganizationYou',
    full: {
        canAddMembers: true,
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        canView: true,
        isStarred: true,
        isViewed: true,
        yourMembership: {
            id: true,
            created_at: true,
            updated_at: true,
            isAdmin: true,
            permissions: true,
        }
    },
}

export const organizationPartial: GqlPartial<Organization> = {
    __typename: 'Organization',
    full: {
        id: true,
        created_at: true,
        handle: true,
        isOpenToNewMembers: true,
        isPrivate: true,
        stars: true,
        reportsCount: true,
        resourceList: resourcePartial.list,
        roles: {
            id: true,
            created_at: true,
            updated_at: true,
            name: true,
            translations: {
                id: true,
                language: true,
                description: true,
            },
        },
        tags: tagPartial.list,
        translations: organizationTranslationPartial.full,
        you: organizationYouPartial.full,
    },
    list: {
        id: true,
        commentsCount: true,
        handle: true,
        stars: true,
        isOpenToNewMembers: true,
        isPrivate: true,
        membersCount: true,
        reportsCount: true,
        tags: tagPartial.list,
        translations: organizationTranslationPartial.full,
        you: organizationYouPartial.full,
    },
    name: {
        id: true,
        handle: true,
        translatedName: true,
    }
}