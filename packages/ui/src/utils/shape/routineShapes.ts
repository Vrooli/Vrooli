import { RoutineCreateInput, RoutineTranslationCreateInput, RoutineTranslationUpdateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { Routine, RoutineTranslation, ShapeWrapper } from "types";
import { hasObjectChanged, InputShape, NodeLinkShape, NodeShape, ObjectType, OutputShape, ResourceListShape, shapeInputsCreate, shapeInputsUpdate, shapeNodeLinksCreate, shapeNodeLinksUpdate, shapeNodesCreate, shapeNodesUpdate, shapeOutputsCreate, shapeOutputsUpdate, shapeResourceListsCreate, shapeResourceListsUpdate, shapeTagsCreate, shapeTagsUpdate, TagShape } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type RoutineTranslationShape = Omit<ShapeWrapper<RoutineTranslation>, 'language' | 'instructions' | 'title'> & {
    id: string;
    language: RoutineTranslationCreateInput['language'];
    instructions: RoutineTranslationCreateInput['instructions'];
    title: RoutineTranslationCreateInput['title'];
}

export const shapeRoutineTranslationCreate = (item: RoutineTranslationShape): RoutineTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    instructions: item.instructions,
    title: item.title,
})

export const shapeRoutineTranslationUpdate = (
    original: RoutineTranslationShape,
    updated: RoutineTranslationShape
): RoutineTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        instructions: u.instructions !== o.instructions ? u.instructions : undefined,
        title: u.title !== o.title ? u.title : undefined,
    }))

export const shapeRoutineTranslationsCreate = (items: RoutineTranslationShape[] | null | undefined): {
    translationsCreate?: RoutineTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeRoutineTranslationCreate);

export const shapeRoutineTranslationsUpdate = (
    o: RoutineTranslationShape[] | null | undefined,
    u: RoutineTranslationShape[] | null | undefined
): {
    translationsCreate?: RoutineTranslationCreateInput[],
    translationsUpdate?: RoutineTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeRoutineTranslationCreate, shapeRoutineTranslationUpdate)

export type RoutineShape = Omit<ShapeWrapper<Routine>, 'complexity' | 'simplicity' | 'inputs' | 'nodeLinks' | 'nodes' | 'outputs' | 'resourceLists' | 'runs' | 'tags'> & {
    id: string;
    inputs: InputShape[];
    nodeLinks?: NodeLinkShape[];
    nodes?: Omit<NodeShape, 'routineId'>[];
    outputs?: OutputShape[];
    owner?: {
        __typename: 'User' | 'Organization';
        id: string;
    } | null;
    parent?: {
        id: string
    } | null;
    resourceLists?: ResourceListShape[];
    tags?: TagShape[];
    translations: RoutineTranslationShape[];
}

export const shapeRoutineCreate = (item: RoutineShape): RoutineCreateInput => ({
    id: item.id,
    isAutomatable: item.isAutomatable,
    isComplete: item.isComplete,
    isInternal: item.isInternal,
    version: item.version,
    parentId: item.parent?.id,
    // projectId: item.p
    createdByUserId: item.owner?.__typename === ObjectType.User ? item.owner.id : undefined,
    createdByOrganizationId: item.owner?.__typename === ObjectType.Organization ? item.owner.id : undefined,
    ...shapeNodesCreate(item.nodes?.map(n => ({ ...n, routineId: item.id }))),
    ...shapeNodeLinksCreate(item.nodeLinks),
    ...shapeInputsCreate(item.inputs),
    ...shapeOutputsCreate(item.outputs),
    ...shapeRoutineTranslationsCreate(item.translations),
    ...shapeResourceListsCreate(item.resourceLists),
    ...shapeTagsCreate(item.tags ?? []),
})

export const shapeRoutineUpdate = (
    original: RoutineShape,
    updated: RoutineShape
): RoutineUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        isAutomatable: u.isAutomatable,
        isComplete: u.isComplete,
        isInternal: u.isInternal,
        version: u.version,
        parentId: u.parent?.id,
        // projectId: u.p
        userId: u.owner?.__typename === ObjectType.User ? u.owner.id : undefined,
        organizationId: u.owner?.__typename === ObjectType.Organization ? u.owner.id : undefined,
        ...shapeNodesUpdate(o.nodes?.map(n => ({ ...n, routineId: o.id })), u.nodes?.map(n => ({ ...n, routineId: o.id }))),
        ...shapeNodeLinksUpdate(o.nodeLinks, u.nodeLinks),
        ...shapeInputsUpdate(o.inputs, u.inputs),
        ...shapeOutputsUpdate(o.outputs, u.outputs),
        ...shapeRoutineTranslationsUpdate(o.translations, u.translations),
        ...shapeResourceListsUpdate(o.resourceLists, u.resourceLists),
        ...shapeTagsUpdate(o.tags, u.tags),
    }))

export const shapeRoutinesCreate = (items: RoutineShape[] | null | undefined): {
    routinesCreate?: RoutineCreateInput[],
} => shapeCreateList(items, 'routines', shapeRoutineCreate);

export const shapeRoutinesUpdate = (
    o: RoutineShape[] | null | undefined,
    u: RoutineShape[] | null | undefined
): {
    routinesCreate?: RoutineCreateInput[],
    routinesUpdate?: RoutineUpdateInput[],
    routinesDelete?: string[],
} => shapeUpdateList(o, u, 'routines', hasObjectChanged, shapeRoutineCreate, shapeRoutineUpdate)