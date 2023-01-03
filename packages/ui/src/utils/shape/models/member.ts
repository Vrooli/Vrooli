import { Member, MemberUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type MemberShape = Pick<Member, 'id'>

export const shapeMember: ShapeModel<MemberShape, null, MemberUpdateInput> = {
    update: (o, u) => ({}) as any
}