import { MemberInvite, MemberInviteCreateInput, MemberInviteUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { OrganizationShape } from "./organization";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type MemberInviteShape = Pick<MemberInvite, "id" | "message" | "willBeAdmin" | "willHavePermissions"> & {
    __typename: "MemberInvite";
    organization: CanConnect<OrganizationShape>;
    user: { __typename: "User", id: string };
}

export const shapeMemberInvite: ShapeModel<MemberInviteShape, MemberInviteCreateInput, MemberInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message", "willBeAdmin", "willHavePermissions"),
        ...createRel(d, "organization", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message", "willBeAdmin", "willHavePermissions"),
    }, a),
};
