import { Role, RoleTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const roleTranslationPartial: GqlPartial<RoleTranslation> = {
    __typename: 'RoleTranslation',
    common: {
        id: true,
        language: true,
        description: true,
    },
    full: {},
    list: {},
}

export const rolePartial: GqlPartial<Role> = {
    __typename: 'Role',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        permissions: true,
        membersCount: true,
        organization: async () => relPartial((await import('./organization')).organizationPartial, 'nav', { omit: 'roles' }),
        translations: () => relPartial(roleTranslationPartial, 'full'),
    },
    full: {
        members: async () => relPartial((await import('./member')).memberPartial, 'nav', { omit: 'reminder' }),
    },
    list: {},
}