import { MeetingInvite, MeetingInviteCreateInput, MeetingInviteUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type MeetingInviteShape = Pick<MeetingInvite, 'id'> & {
    __typename?: 'MeetingInvite';
}

export const shapeMeetingInvite: ShapeModel<MeetingInviteShape, MeetingInviteCreateInput, MeetingInviteUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}