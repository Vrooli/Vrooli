import { Organization, OrganizationTranslation, OrganizationYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const organizationTranslationPartial: GqlPartial<OrganizationTranslation> = {
    __typename: 'OrganizationTranslation',
    common: {
        id: true,
        language: true,
        bio: true,
        name: true,
    },
    full: {},
    list: {},
}

export const organizationYouPartial: GqlPartial<OrganizationYou> = {
    __typename: 'OrganizationYou',
    common: {
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
    full: {},
    list: {},
}

export const organizationPartial: GqlPartial<Organization> = {
    __typename: 'Organization',
    common: {
        __define: {
            0: [require('./tag').tagPartial, 'list'],
        },
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
        tags: { __use: 0 },
        translations: () => relPartial(organizationTranslationPartial, 'full'),
        you: () => relPartial(organizationYouPartial, 'full'),
    },
    full: {
        roles: () => relPartial(require('./role').rolePartial, 'full', { omit: 'organization' }),
    },
    list: { },
    nav: {
        id: true,
        handle: true,
        you: () => relPartial(organizationYouPartial, 'full'),
    }
}