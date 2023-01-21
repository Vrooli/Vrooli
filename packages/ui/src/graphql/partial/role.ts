import { Role, RoleTranslation } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { memberPartial } from "./member";
import { organizationPartial } from "./organization";

export const roleTranslationPartial: GqlPartial<RoleTranslation> = {
    __typename: 'RoleTranslation',
    full: {
        id: true,
        language: true,
        description: true,
    },
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
        organization: () => relPartial(organizationPartial, 'nav', { omit: 'roles' }),
    },
    full: {
        members: () => relPartial(memberPartial, 'nav', { omit: 'reminder' }),
        translations: () => relPartial(roleTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(roleTranslationPartial, 'list'),
    }
}