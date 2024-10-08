import { Note, NoteCreateInput, NoteUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { LabelShape, shapeLabel } from "./label";
import { NoteVersionShape, shapeNoteVersion } from "./noteVersion";
import { TagShape, shapeTag } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";

export type NoteShape = Pick<Note, "id" | "isPrivate"> & {
    __typename: "Note";
    labels?: CanConnect<LabelShape>[] | null;
    owner: OwnerShape | null | undefined;
    parent?: CanConnect<NoteVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<NoteVersionShape, "root">[] | null;
}

export const shapeNote: ShapeModel<NoteShape, NoteCreateInput, NoteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeNoteVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeNoteVersion),
    }),
};
