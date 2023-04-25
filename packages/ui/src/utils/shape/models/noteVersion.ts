import { NoteVersion, NoteVersionCreateInput, NoteVersionTranslation, NoteVersionTranslationCreateInput, NoteVersionTranslationUpdateInput, NoteVersionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { NoteShape, shapeNote } from "./note";
import { ResourceListShape } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NoteVersionTranslationShape = Pick<NoteVersionTranslation, 'id' | 'language' | 'description' | 'name' | 'text'> & {
    __typename?: 'NoteVersionTranslation';
}

export type NoteVersionShape = Pick<NoteVersion, 'id' | 'isPrivate' | 'versionLabel' | 'versionNotes'> & {
    __typename?: 'NoteVersion';
    directoryListings?: { id: string }[] | null;
    resourceList?: { id: string } | ResourceListShape | null;
    root?: { id: string } | NoteShape | null;
    translations?: NoteVersionTranslationShape[] | null;
}

export const shapeNoteVersionTranslation: ShapeModel<NoteVersionTranslationShape, NoteVersionTranslationCreateInput, NoteVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'name', 'text'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'language', 'description', 'name', 'text'), a)
}

export const shapeNoteVersion: ShapeModel<NoteVersionShape, NoteVersionCreateInput, NoteVersionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isPrivate', 'versionLabel', 'versionNotes'),
        ...createRel(d, 'directoryListings', ['Connect'], 'many'),
        ...createRel(d, 'root', ['Connect', 'Create'], 'one', shapeNote),
        ...createRel(d, 'translations', ['Create'], 'many', shapeNoteVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isPrivate', 'versionLabel', 'versionNotes'),
        ...updateRel(o, u, 'directoryListings', ['Connect', 'Disconnect'], 'many'),
        ...updateRel(o, u, 'root', ['Update'], 'one', shapeNote),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeNoteVersionTranslation),
    }, a)
}