import { ApiVersion, ApiVersionCreateInput, ApiVersionTranslation, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput, ApiVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ApiShape, shapeApi } from "./api";
import { shapeResourceList } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ApiVersionTranslationShape = Pick<ApiVersionTranslation, 'id' | 'language' | 'details' | 'name' | 'summary'> & {
    __typename?: 'ApiVersionTranslation';
}

export type ApiVersionShape = Pick<ApiVersion, 'id' | 'callLink' | 'documentationLink' | 'isPrivate' | 'versionLabel' | 'versionNotes'> & {
    __typename?: 'ApiVersion';
    directoryListings?: { id: string }[] | null;
    resourceList?: { id: string } | null;
    root?: { id: string } | ApiShape | null;
    translations?: ApiVersionTranslationShape[] | null;
}

export const shapeApiVersionTranslation: ShapeModel<ApiVersionTranslationShape, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'details', 'name', 'summary'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'details', 'summary'), a)
}

export const shapeApiVersion: ShapeModel<ApiVersionShape, ApiVersionCreateInput, ApiVersionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'callLink', 'documentationLink', 'isPrivate', 'versionLabel', 'versionNotes'),
        ...createRel(d, 'directoryListings', ['Connect'], 'many'),
        ...createRel(d, 'resourceList', ['Create'], 'one', shapeResourceList),
        ...createRel(d, 'root', ['Connect', 'Create'], 'one', shapeApi),
        ...createRel(d, 'translations', ['Create'], 'many', shapeApiVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'callLink', 'documentationLink', 'isPrivate', 'versionLabel', 'versionNotes'),
        ...updateRel(o, u, 'directoryListings', ['Connect', 'Disconnect'], 'many'),
        ...updateRel(o, u, 'resourceList', ['Create', 'Update'], 'one', shapeResourceList),
        ...updateRel(o, u, 'root', ['Update'], 'one', shapeApi),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeApiVersionTranslation),
    }, a)
}