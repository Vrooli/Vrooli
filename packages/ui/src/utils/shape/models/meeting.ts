import { Meeting, MeetingCreateInput, MeetingUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type MeetingShape = Pick<Meeting, 'id'>

export const shapeMeeting: ShapeModel<MeetingShape, MeetingCreateInput, MeetingUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}