import { ApiVersion, ApiVersionCreateInput, ApiVersionTranslation, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput, ApiVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ApiShape, shapeApi } from "./api";
import { shapeResourceList } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ApiVersionTranslationShape = Pick<ApiVersionTranslation, 'id' | 'language' | 'details' | 'summary'>

export type ApiVersionShape = Pick<ApiVersion, 'id' | 'callLink' | 'documentationLink' | 'isLatest' | 'versionIndex' | 'versionLabel' | 'versionNotes'> & {
    directoryListings?: { id: string }[];
    resourceList?: { id: string };
    root?: { id: string } | ApiShape;
    translations?: ApiVersionTranslationShape[];
}

export const shapeApiVersionTranslation: ShapeModel<ApiVersionTranslationShape, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'details', 'summary'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'details', 'summary'))
}

export const shapeApiVersion: ShapeModel<ApiVersionShape, ApiVersionCreateInput, ApiVersionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'callLink', 'documentationLink', 'isLatest', 'isPrivate', 'versionIndex', 'versionLabel', 'versionNotes'),
        ...createRel(d, 'directoryListings', ['Connect'], 'many'),
        ...createRel(d, 'resourceList', ['Create'], 'one', shapeResourceList),
        ...createRel(d, 'root', ['Connect', 'Create'], 'one', shapeApi),
        ...createRel(d, 'translations', ['Create'], 'many', shapeApiVersionTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'callLink', 'documentationLink', 'isLatest', 'isPrivate', 'versionIndex', 'versionLabel', 'versionNotes'),
        ...updateRel(o, u, 'directoryListings', ['Connect', 'Disconnect'], 'many'),
        ...updateRel(o, u, 'resourceList', ['Create', 'Update'], 'one', shapeResourceList),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeApiVersionTranslation),
    })
}