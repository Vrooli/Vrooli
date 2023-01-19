import { RoleTranslation } from "@shared/consts";
import { GqlPartial } from "types";

export const roleTranslationPartial: GqlPartial<RoleTranslation> = {
    __typename: 'RoleTranslation',
    full: () => ({
        id: true,
        language: true,
        description: true,
    }),
}

export const listRoleFields = ['Role', `{
    id
}`] as const;
export const roleFields = ['Role', `{
    id
}`] as const;