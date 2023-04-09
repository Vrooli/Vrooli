import { MemberInvite, MemberInviteCreateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type MemberInviteShape = Pick<MemberInvite, 'id'> & {
    __typename?: 'MemberInvite';
}

export const shapeMemberInvite: ShapeModel<MemberInviteShape, MemberInviteCreateInput, null> = {
    create: (d) => ({}) as any
}