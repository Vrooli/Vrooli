import { Organization, OrganizationTranslation, OrganizationYou } from "@shared/consts";
import { relPartial } from '../utils';
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
    common: {
        id: true,
        handle: true,
        created_at: true,
        updated_at: true,
        isOpenToNewMembers: true,
        isPrivate: true,
        commentsCount: true,
        membersCount: true,
        reportsCount: true,
        stars: true,
        tags: () => relPartial(require('./tag').tagPartial, 'list'),
    },
    full: {
        roles: () => relPartial(require('./role').rolePartial, 'full', { omit: 'organization' }),
        translations: () => relPartial(organizationTranslationPartial, 'full'),
        you: () => relPartial(organizationYouPartial, 'full'),
    },
    list: {
        roles: () => relPartial(require('./role').rolePartial, 'list', { omit: 'organization' }),
        translations: () => relPartial(organizationTranslationPartial, 'full'),
        you: () => relPartial(organizationYouPartial, 'full'),
    },
    nav: {
        id: true,
        handle: true,
        translations: () => relPartial(organizationTranslationPartial, 'list'),
        you: () => relPartial(organizationYouPartial, 'full'),
    }
}