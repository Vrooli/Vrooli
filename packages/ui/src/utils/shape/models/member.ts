import { Member, MemberUpdateInput, User } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate, updatePrims } from "./tools";

export type MemberShape = Pick<Member, "id" | "isAdmin" | "permissions"> & {
    __typename: "Member";
    user: Pick<User, "updated_at" | "handle" | "id" | "isBot" | "name" | "profileImage">;
}

export const shapeMember: ShapeModel<MemberShape, null, MemberUpdateInput> = {
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isAdmin", "permissions"),
    }, a),
};
