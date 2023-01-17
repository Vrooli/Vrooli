import { MemberInvite, MemberInviteCreateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type MemberInviteShape = Pick<MemberInvite, 'id'>

export const shapeMemberInvite: ShapeModel<MemberInviteShape, MemberInviteCreateInput, null> = {
    create: (d) => ({}) as any
}