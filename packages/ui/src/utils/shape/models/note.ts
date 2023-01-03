import { Note, NoteCreateInput, NoteUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type NoteShape = Pick<Note, 'id'>

export const shapeNote: ShapeModel<NoteShape, NoteCreateInput, NoteUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}