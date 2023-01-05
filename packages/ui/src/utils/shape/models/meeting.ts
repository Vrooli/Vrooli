import { Meeting, MeetingCreateInput, MeetingTranslation, MeetingTranslationCreateInput, MeetingTranslationUpdateInput, MeetingUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type MeetingTranslationShape = Pick<MeetingTranslation, 'id' | 'language' | 'description' | 'link' | 'name'>

export type MeetingShape = Pick<Meeting, 'id'>

export const shapeMeetingTranslation: ShapeModel<MeetingTranslationShape, MeetingTranslationCreateInput, MeetingTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'link', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'link', 'name'))
}

export const shapeMeeting: ShapeModel<MeetingShape, MeetingCreateInput, MeetingUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}