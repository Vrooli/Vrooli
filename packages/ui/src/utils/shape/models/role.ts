import { Role, RoleCreateInput, RoleTranslation, RoleTranslationCreateInput, RoleTranslationUpdateInput, RoleUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type RoleTranslationShape = Pick<RoleTranslation, 'id' | 'language' | 'description'> & {
    __typename?: 'RoleTranslation';
}

export type RoleShape = Pick<Role, 'id' | 'name' | 'permissions'> & {
    __typename?: 'Role';
    members?: { id: string }[] | null;
    organization: { id: string };
    translations?: RoleTranslationShape[] | null;
}

export const shapeRoleTranslation: ShapeModel<RoleTranslationShape, RoleTranslationCreateInput, RoleTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description'))
}

export const shapeRole: ShapeModel<RoleShape, RoleCreateInput, RoleUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'name', 'permissions'),
        ...createRel(d, 'members', ['Connect'], 'many'),
        ...createRel(d, 'organization', ['Connect'], 'one'),
        ...createRel(d, 'translations', ['Create'], 'many', shapeRoleTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'name', 'permissions'),
        ...updateRel(o, u, 'members', ['Connect', 'Disconnect'], 'many'),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeRoleTranslation),
    })
}