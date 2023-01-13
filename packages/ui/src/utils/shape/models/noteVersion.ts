import { NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type NoteVersionShape = Pick<NoteVersion, 'id'>

export const shapeNoteVersion: ShapeModel<NoteVersionShape, NoteVersionCreateInput, NoteVersionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}