import { Organization, OrganizationTranslation, OrganizationYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const organizationTranslation: GqlPartial<OrganizationTranslation> = {
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

export const organizationYou: GqlPartial<OrganizationYou> = {
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

export const organization: GqlPartial<Organization> = {
    __typename: 'Organization',
    common: {
        __define: {
            0: async () => rel((await import('./tag')).tag, 'list'),
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
        translations: () => rel(organizationTranslation, 'full'),
        you: () => rel(organizationYou, 'full'),
    },
    full: {
        roles: async () => rel((await import('./role')).role, 'full', { omit: 'organization' }),
    },
    list: { },
    nav: {
        id: true,
        handle: true,
        you: () => rel(organizationYou, 'full'),
    }
}