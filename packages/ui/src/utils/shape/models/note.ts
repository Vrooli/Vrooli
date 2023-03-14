import { Note, NoteCreateInput, NoteUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { NoteVersionShape, createOwner, createPrims, createRel, createVersion, LabelShape, shapeNoteVersion, shapeLabel, shapeTag, shapeUpdate, TagShape, updateOwner, updatePrims, updateRel, updateVersion } from "utils";

export type NoteShape = Pick<Note, 'id' | 'isPrivate'> & {
    labels?: ({ id: string } | LabelShape)[];
    owner: { __typename: 'User' | 'Organization', id: string } | null;
    parent?: { id: string } | null;
    tags?: ({ tag: string } | TagShape)[];
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionInfo?: NoteVersionShape | null;
}

export const shapeNote: ShapeModel<NoteShape, NoteCreateInput, NoteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isPrivate'),
        ...createOwner(d, 'ownedBy'),
        ...createRel(d, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(d, 'parent', ['Connect'], 'one'),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createVersion(d, shapeNoteVersion, (v) => ({ ...v, root: { id: d.id } })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isPrivate'),
        ...updateOwner(o, u, 'ownedBy'),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateVersion(o, u, shapeNoteVersion, (v, i) => ({ ...v, root: { id: i.id } })),
    }, a)
}