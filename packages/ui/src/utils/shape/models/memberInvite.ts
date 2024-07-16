import { MemberInvite, MemberInviteCreateInput, MemberInviteUpdateInput, User } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { TeamShape } from "./team";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type MemberInviteShape = Pick<MemberInvite, "id" | "message" | "willBeAdmin" | "willHavePermissions"> & {
    __typename: "MemberInvite";
    team: CanConnect<TeamShape>;
    user: Pick<User, "__typename" | "updated_at" | "handle" | "id" | "isBot" | "name" | "profileImage">;
}

export const shapeMemberInvite: ShapeModel<MemberInviteShape, MemberInviteCreateInput, MemberInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message", "willBeAdmin", "willHavePermissions"),
        ...createRel(d, "team", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message", "willBeAdmin", "willHavePermissions"),
    }, a),
};
