import { Role, RoleCreateInput, RoleTranslation, RoleTranslationCreateInput, RoleTranslationUpdateInput, RoleUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { MemberShape } from "./member";
import { TeamShape } from "./team";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type RoleTranslationShape = Pick<RoleTranslation, "id" | "language" | "description"> & {
    __typename?: "RoleTranslation";
}

export type RoleShape = Pick<Role, "id" | "name" | "permissions"> & {
    __typename: "Role";
    members?: CanConnect<MemberShape>[] | null;
    team: CanConnect<TeamShape>;
    translations?: RoleTranslationShape[] | null;
}

export const shapeRoleTranslation: ShapeModel<RoleTranslationShape, RoleTranslationCreateInput, RoleTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description")),
};

export const shapeRole: ShapeModel<RoleShape, RoleCreateInput, RoleUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "permissions"),
        ...createRel(d, "members", ["Connect"], "many"),
        ...createRel(d, "team", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeRoleTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "permissions"),
        ...updateRel(o, u, "members", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoleTranslation),
    }),
};
