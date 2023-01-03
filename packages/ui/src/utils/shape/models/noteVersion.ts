import { NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type NoteVersionShape = Pick<NoteVersion, 'id'>

export const shapeNoteVersion: ShapeModel<NoteVersionShape, NoteVersionCreateInput, NoteVersionUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}