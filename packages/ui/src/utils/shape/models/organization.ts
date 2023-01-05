import { Organization, OrganizationCreateInput, OrganizationTranslation, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput, OrganizationUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, MemberInviteShape, ResourceListShape, RoleShape, shapeMemberInvite, shapeResourceList, shapeRole, shapeTag, shapeUpdate, TagShape, updatePrims, updateRel } from "utils";

export type OrganizationTranslationShape = Pick<OrganizationTranslation, 'id' | 'language' | 'bio' | 'name'>

export type OrganizationShape = Pick<Organization, 'id' | 'handle' | 'isOpenToNewMembers' | 'isPrivate'> & {
    memberInvites?: MemberInviteShape[];
    membersDelete?: { id: string }[];
    resourceList?: ResourceListShape;
    roles?: RoleShape[];
    tags?: ({ tag: string } | TagShape)[];
    translations?: OrganizationTranslationShape[];
}

export const shapeOrganizationTranslation: ShapeModel<OrganizationTranslationShape, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'bio', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'bio', 'name'))
}

export const shapeOrganization: ShapeModel<OrganizationShape, OrganizationCreateInput, OrganizationUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'handle', 'isOpenToNewMembers', 'isPrivate'),
        ...createRel(d, 'memberInvites', ['Create'], 'many', shapeMemberInvite),
        ...createRel(d, 'resourceList', ['Create'], 'one', shapeResourceList),
        ...createRel(d, 'roles', ['Create'], 'many', shapeRole),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createRel(d, 'translations', ['Create'], 'many', shapeOrganizationTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'handle', 'isOpenToNewMembers', 'isPrivate'),
        ...updateRel(o, u, 'memberInvites', ['Create', 'Delete'], 'many', shapeMemberInvite),
        ...updateRel(o, u, 'resourceList', ['Create', 'Update'], 'one', shapeResourceList),
        ...updateRel(o, u, 'roles', ['Create', 'Update', 'Delete'], 'many', shapeRole),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeOrganizationTranslation),
        ...(u.membersDelete ? { membersDelete: u.membersDelete.map(id => ({ id })) } : {}),
    })
}