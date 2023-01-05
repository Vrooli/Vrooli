import { Member, MemberUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type MemberShape = Pick<Member, 'id'>

export const shapeMember: ShapeModel<MemberShape, null, MemberUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {}) as any
}