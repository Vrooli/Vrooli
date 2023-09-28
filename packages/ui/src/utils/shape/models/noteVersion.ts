import { NotePage, NotePageCreateInput, NotePageUpdateInput, NoteVersion, NoteVersionCreateInput, NoteVersionTranslation, NoteVersionTranslationCreateInput, NoteVersionTranslationUpdateInput, NoteVersionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { NoteShape, shapeNote } from "./note";
import { ResourceListShape } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type NotePageShape = Pick<NotePage, "id" | "pageIndex" | "text"> & {
    __typename?: "NotePage";
}

export type NoteVersionTranslationShape = Pick<NoteVersionTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "NoteVersionTranslation";
    pages?: NotePageShape[] | null;
}

export type NoteVersionShape = Pick<NoteVersion, "id" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename: "NoteVersion";
    directoryListings?: { id: string }[] | null;
    resourceList?: { id: string } | ResourceListShape | null;
    root?: { id: string } | NoteShape | null;
    translations?: NoteVersionTranslationShape[] | null;
}

export const shapeNotePage: ShapeModel<NotePageShape, NotePageCreateInput, NotePageUpdateInput> = {
    create: (d) => createPrims(d, "id", "pageIndex", "text"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "pageIndex", "text"), a),
};

export const shapeNoteVersionTranslation: ShapeModel<NoteVersionTranslationShape, NoteVersionTranslationCreateInput, NoteVersionTranslationUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "language", "description", "name"),
        ...createRel(d, "pages", ["Create"], "many", shapeNotePage),
    }),
    update: (o, u, a) => shapeUpdate(u, ({
        ...updateTranslationPrims(o, u, "id", "language", "description", "name"),
        ...updateRel(o, u, "pages", ["Create", "Update", "Delete"], "many", shapeNotePage),
    }), a),
};

export const shapeNoteVersion: ShapeModel<NoteVersionShape, NoteVersionCreateInput, NoteVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Connect"], "many"),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeNote, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "translations", ["Create"], "many", shapeNoteVersionTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "root", ["Update"], "one", shapeNote),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNoteVersionTranslation),
    }, a),
};
