import { GqlModelType, lowercaseFirstLetter } from "@local/shared";
import { CustomError } from "../events";
import { ObjectMap } from "../models/base";
import { PreMap } from "../models/types";
import { PrismaType, SessionUserToken } from "../types";
import { shapeRelationshipData } from "./shapeRelationshipData";
import { RelationshipType, RelConnect, RelCreate, RelDelete, RelDisconnect, RelUpdate } from "./types";

// Array if isOneToOne is false, otherwise single
type MaybeArray<T, IsOneToOne extends boolean> =
    IsOneToOne extends true ? T : T[];

// Array if isOneToOne is false, otherwise boolean
type MaybeArrayBoolean<T, IsOneToOne extends boolean> =
    IsOneToOne extends true ? boolean : T[];

// [{ where: { id: 'id' }, data: { ... } }] if isOneToOne is false, otherwise { ... }
type MaybeArrayUpdate<T extends RelUpdate<any, string>, IsOneToOne extends boolean> =
    IsOneToOne extends true ? T["data"] : T[];

type ShapeHelperOptionalInput<
    IsOneToOne extends boolean,
    RelFields extends string,
    FieldName extends string,
> = (
        ({ [x in `${FieldName}Delete`]?: "Delete" extends RelFields ? MaybeArrayBoolean<string, IsOneToOne> | null | undefined : never }) &
        ({ [x in `${FieldName}Disconnect`]?: "Disconnect" extends RelFields ? MaybeArrayBoolean<string, IsOneToOne> | null | undefined : never }) &
        ({ [x in `${FieldName}Update`]?: "Update" extends RelFields ? MaybeArray<any, IsOneToOne> | null | undefined : never })
    )

export type ShapeHelperInput<
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    RelFields extends string,
    FieldName extends string,
> = IsRequired extends true ?
    // If required, ensure that at least connect or create is present
    "Connect" extends RelFields ?
    "Create" extends RelFields ? ((
        ({ [x in `${FieldName}Connect`]?: MaybeArray<string, IsOneToOne> }) &
        ({ [x in `${FieldName}Create`]?: MaybeArray<any, IsOneToOne> | null | undefined })
    ) | (
        ({ [x in `${FieldName}Connect`]?: MaybeArray<string, IsOneToOne> | null | undefined }) &
        ({ [x in `${FieldName}Create`]?: MaybeArray<any, IsOneToOne> })
    ) & ShapeHelperOptionalInput<IsOneToOne, RelFields[number], FieldName>) : (
        { [x in `${FieldName}Connect`]?: MaybeArray<string, IsOneToOne> }
    ) & ShapeHelperOptionalInput<IsOneToOne, RelFields[number], FieldName> :
    "Create" extends RelFields ? (
        { [x in `${FieldName}Create`]?: MaybeArray<any, IsOneToOne> }
    ) & ShapeHelperOptionalInput<IsOneToOne, RelFields[number], FieldName> :
    ShapeHelperOptionalInput<IsOneToOne, RelFields[number], FieldName> : (
        ({ [x in `${FieldName}Connect`]?: MaybeArray<string, IsOneToOne> | null | undefined }) &
        ({ [x in `${FieldName}Create`]?: MaybeArray<any, IsOneToOne> | null | undefined })
    ) & ShapeHelperOptionalInput<IsOneToOne, RelFields[number], FieldName>

type ShapeHelperOptionalOutput<
    IsOneToOne extends boolean,
    RelFields extends string,
    PrimaryKey extends string,
> = {
    delete?: "Delete" extends RelFields ? MaybeArrayBoolean<RelDelete<PrimaryKey>, IsOneToOne> | undefined : never,
    disconnect?: "Disconnect" extends RelFields ? MaybeArrayBoolean<RelDisconnect<PrimaryKey>, IsOneToOne> | undefined : never,
    update?: "Update" extends RelFields ? MaybeArrayUpdate<RelUpdate<any, PrimaryKey>, IsOneToOne> | undefined : never,
}

type WrapInField<T, FieldName extends string> = {
    [x in FieldName]: T
}

export type ShapeHelperOutput<
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    RelFields extends string,
    FieldName extends string,
    PrimaryKey extends string,
> = WrapInField<(IsRequired extends true ?
    // If required, ensure that at least connect or create is present
    "Connect" extends RelFields ?
    "Create" extends RelFields ? ({
        connect: MaybeArray<RelConnect<PrimaryKey>, IsOneToOne>,
        create?: MaybeArray<RelCreate<any>, IsOneToOne> | undefined,
    } | {
        connect?: MaybeArray<RelConnect<PrimaryKey>, IsOneToOne> | undefined,
        create: MaybeArray<RelCreate<any>, IsOneToOne>,
    } & ShapeHelperOptionalOutput<IsOneToOne, RelFields, PrimaryKey>) : ({
        connect: MaybeArray<RelConnect<PrimaryKey>, IsOneToOne>,
    } & ShapeHelperOptionalOutput<IsOneToOne, RelFields, PrimaryKey>) :
    "Create" extends RelFields ? ({
        create: MaybeArray<RelCreate<any>, IsOneToOne>,
    } & ShapeHelperOptionalOutput<IsOneToOne, RelFields, PrimaryKey>) :
    ShapeHelperOptionalOutput<IsOneToOne, RelFields, PrimaryKey> : ({
        connect?: "Connect" extends RelFields ? MaybeArray<RelConnect<PrimaryKey>, IsOneToOne> | undefined : never,
        create?: "Create" extends RelFields ? MaybeArray<RelCreate<any>, IsOneToOne> | undefined : never,
    } & ShapeHelperOptionalOutput<IsOneToOne, RelFields, PrimaryKey>)
), FieldName>

export type ShapeHelperProps<
    Input extends ShapeHelperInput<IsOneToOne, IsRequired, Types[number], RelField>,
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    Types extends readonly RelationshipType[],
    RelField extends string,
    PrimaryKey extends string,
    SoftDelete extends boolean,
> = {
    /** The data to convert */
    data: Input,
    /**
     * True if relationship is one-to-one. This makes 
     * the results a single object instead of an array
     */
    isOneToOne: IsOneToOne,
    /** True if relationship must be provided */
    isRequired: IsRequired,
    /**
    * If relationship is a join table, data required to create the join table record
    * 
    * NOTE: Does not differentiate between a disconnect and a delete. How these are handled is 
    * determined by the database cascading.
    */
    joinData?: {
        fieldName: string, // e.g. organization.tags.tag => 'tag'
        uniqueFieldName: string, // e.g. organization.tags.tag => 'organization_tags_taggedid_tagTag_unique'
        childIdFieldName: string, // e.g. organization.tags.tag => 'tagTag'
        parentIdFieldName: string, // e.g. organization.tags.tag => 'taggedId'
        parentId: string | null, // Only needed if not a create
    }
    objectType: `${GqlModelType}`,
    /**
    * The name of the parent relationship, from the child's perspective. 
    * Used to ensure that the child does not have a circular reference to the parent.
    */
    parentRelationshipName: string,
    /**
     * A map of pre-shape data for all objects in the mutation, keyed by object type. 
     */
    preMap: PreMap,
    /** The name of the primaryKey key field. Defaults to "id" */
    primaryKey?: PrimaryKey,
    /** The Prisma client */
    prisma: PrismaType,
    relation: RelField,
    relTypes: Types,
    /**
     * If true, relationship is set to "isDelete" 
     * true instead of actually deleting the record
     */
    softDelete?: SoftDelete,
    /**
     * Session data of the user performing the operation. Relationship building is only used when performing 
     * create, update, and delete operations, so id is always required
     */
    userData: SessionUserToken,
}
/**
 * Creates the relationship operations for a mutater shape create or update function
 * @param data The GraphQL update data object
 * @param relTypes The allowed operations on the relations (e.g. create, connect)
 * @param relation The name of the relationship field
 * @param isRequired Whether the connection is required
 * @param isOneToOne Whether the connection is a single object or an array
 * @returns An object with the connections
 */
export const shapeHelper = async<
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    Types extends readonly RelationshipType[],
    RelField extends string,
    PrimaryKey extends string,
    SoftDelete extends boolean,
    Input extends ShapeHelperInput<IsOneToOne, IsRequired, Types[number], RelField>,
>({
    data,
    isOneToOne,
    isRequired,
    joinData,
    objectType,
    parentRelationshipName,
    preMap,
    primaryKey = "id" as PrimaryKey,
    prisma,
    relation,
    relTypes,
    softDelete = false as SoftDelete,
    userData,
}: ShapeHelperProps<Input, IsOneToOne, IsRequired, Types, RelField, PrimaryKey, SoftDelete>):
    Promise<ShapeHelperOutput<IsOneToOne, IsRequired, Types[number], RelField, PrimaryKey>> => {
    // Initialize result
    let result: { [x: string]: any } = {};
    // If both connect and create do not exist, and it's required, throw an error
    if (!data[`${relation}Connect` as string] && !data[`${relation}Create` as string] && isRequired) {
        throw new CustomError("0368", "InvalidArgs", ["en"], { relation, data });
    }
    // Loop through relation types, and convert all to a Prisma-shaped array
    for (const t of relTypes) {
        // If not in data, skip
        const curr = data[`${relation}${t}` as string];
        if (!curr) continue;
        // Shape the data
        const currShaped = shapeRelationshipData(curr);
        // Add to result
        result[lowercaseFirstLetter(t)] = Array.isArray(result[lowercaseFirstLetter(t)]) ?
            [...result[lowercaseFirstLetter(t)] as any, ...currShaped] :
            currShaped;
    }
    // Now we can further shape the result
    // Connects, diconnects, and deletes must be shaped in the form of { id: '123' } (i.e. no other fields)
    if (Array.isArray(result.connect) && result.connect.length > 0) result.connect = result.connect.map((e: { [x: string]: any }) => ({ [primaryKey]: e[primaryKey] }));
    else result.connect = undefined;
    if (Array.isArray(result.disconnect) && result.disconnect.length > 0) result.disconnect = result.disconnect.map((e: { [x: string]: any }) => ({ [primaryKey]: e[primaryKey] }));
    else result.disconnect = undefined;
    if (Array.isArray(result.delete) && result.delete.length > 0) result.delete = result.delete.map((e: { [x: string]: any }) => ({ [primaryKey]: e[primaryKey] }));
    else result.delete = undefined;
    // Updates must be shaped in the form of { where: { id: '123' }, data: {...}}
    if (Array.isArray(result.update) && result.update.length > 0) {
        result.update = result.update.map((e: any) => ({ where: { id: e.id }, data: e }));
    }
    else result.update = undefined;
    // Convert deletes to updates if softDelete is true
    if (softDelete && Array.isArray(result.delete) && result.delete.length > 0) {
        const softDeletes = result.delete.map((e: any) => ({ where: { id: e[primaryKey] }, data: { isDeleted: true } }));
        result.update = Array.isArray(result.update) ? [...result.update, ...softDeletes] : softDeletes;
        delete result.delete;
    }
    // Perform nested shapes for create and update
    const mutate = ObjectMap[objectType]?.mutate;
    if (mutate?.shape.create && Array.isArray(result.create) && result.create.length > 0) {
        const shaped: { [x: string]: any }[] = [];
        for (const create of result.create) {
            const created = await (mutate.shape as any).create({ data: create, preMap, prisma, userData });
            // Exclude parent relationship to prevent circular references
            const { [parentRelationshipName]: _, ...rest } = created;
            shaped.push(rest);
        }
        result.create = shaped;
    }
    if (mutate?.shape.update && Array.isArray(result.update) && result.update.length > 0) {
        const shaped: { [x: string]: any }[] = [];
        for (const update of result.update) {
            const updated = await (mutate.shape as any).update({ data: update.data, preMap, prisma, userData, where: update.where });
            // Exclude parent relationship to prevent circular references
            const { [parentRelationshipName]: _, ...rest } = updated;
            shaped.push({ where: update.where, data: rest });
        }
        result.update = shaped;
    }
    // Handle join table, if applicable
    if (joinData) {
        const resultWithJoin: Record<string, any> = { create: [], update: [], delete: [] };
        if (result.connect) {
            // ex: create: [ { tag: { connect: { id: 'asdf' } } } ] <-- A join table always creates on connects
            for (const id of (result?.connect ?? [])) {
                const curr = { [joinData.fieldName]: { connect: id } };
                resultWithJoin.create.push(curr);
            }
        }
        if (result.disconnect) {
            // delete: [ { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ] <-- A join table always deletes on disconnects
            for (const id of (result?.disconnect ?? [])) {
                const curr = {
                    [joinData.uniqueFieldName]: {
                        [joinData.childIdFieldName]: id[primaryKey],
                        [joinData.parentIdFieldName]: joinData.parentId,
                    },
                };
                resultWithJoin.delete.push(curr);
            }
        }
        if (result.delete) {
            // delete: [ { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ]
            for (const id of (result?.delete ?? [])) {
                const curr = {
                    [joinData.uniqueFieldName]: {
                        [joinData.childIdFieldName]: id[primaryKey],
                        [joinData.parentIdFieldName]: joinData.parentId,
                    },
                };
                resultWithJoin.delete.push(curr);
            }
        }
        if (result.create) {
            // ex: create: [ { tag: { create: { id: 'asdf' } } } ]
            for (const id of (result?.create ?? [])) {
                const curr = { [joinData.fieldName]: { create: id } };
                resultWithJoin.create.push(curr);
            }
        }
        if (result.update) {
            // ex: update: [{ 
            //         where: { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } },
            //         data: { tag: { update: { tag: 'fdas', } } }
            //     }]
            for (const data of (result?.update ?? [])) {
                const curr = {
                    where: { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: data.where[primaryKey], [joinData.parentIdFieldName]: joinData.parentId } },
                    data: { [joinData.fieldName]: { update: data.data } },
                };
                resultWithJoin.update.push(curr);
            }
        }
        result = resultWithJoin;
    }
    // If one-to-one, perform some final checks and remove arrays
    if (isOneToOne) {
        // Perform the following checks:
        // 1. If required:
        //    1.a If adding, must have connect or create
        //    a.b If updating and has a disconnect or delete, must have connect or create
        // 3. Does not have both a connect and create
        // 4. Does not have both a disconnect and delete
        const isAdd = ("create" in result || "connect" in result) && !("delete" in result || "disconnect" in result || "update" in result);
        if (isRequired) {
            if (isAdd && !result.connect && !result.create)
                throw new CustomError("0340", "InvalidArgs", userData.languages, { relation });
            if (!isAdd && (result.disconnect || result.delete) && !result.connect && !result.create)
                throw new CustomError("0341", "InvalidArgs", userData.languages, { relation });
        }
        if (result.connect && result.create)
            throw new CustomError("0342", "InvalidArgs", userData.languages, { relation });
        if (result.disconnect && result.delete)
            throw new CustomError("0343", "InvalidArgs", userData.languages, { relation });
        // Remove arrays
        // one-to-one's disconnect/delete must be true or undefined
        if (result.disconnect) result.disconnect = true;
        if (result.delete) result.delete = true;
        // one-to-one's connect/create must be an object, not an array
        if (Array.isArray(result.connect)) result.connect = result.connect.length ? result.connect[0] : undefined;
        if (Array.isArray(result.create)) result.create = result.create.length ? result.create[0] : undefined;
        // one-to-one's update must not have a where, and must be an object, not an array
        if (Array.isArray(result.update)) result.update = result.update.length ? result.update[0].data : undefined;
    }
    // Remove values that are undefined or empty arrays
    result = Object.keys(result).reduce((acc, key) => {
        if (result[key] === undefined || (Array.isArray(result[key]) && result[key].length === 0)) return acc;
        acc[key] = result[key];
        return acc;
    }, {} as any);
    // If there are keys remaining, return result wrapped in the relation name
    if (Object.keys(result).length) return { [relation]: result } as any;
    // Otherwise, return empty object
    return {} as any;
};

/**
 * Typical relations for create inputs
 */
export const addRels = ["Create", "Connect"] as const;

/**
 * Typical relations for update inputs
 */
export const updateRels = ["Connect", "Create", "Delete", "Disconnect", "Update"] as const;
