import { Note, NoteCreateInput, NoteUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type NoteShape = Pick<Note, 'id'>

export const shapeNote: ShapeModel<NoteShape, NoteCreateInput, NoteUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}