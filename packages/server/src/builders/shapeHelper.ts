import { GqlModelType, lowercaseFirstLetter, uuidValidate } from "@local/shared";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { PreMap } from "../models/types";
import { SessionUserToken } from "../types";
import { IdsCreateToConnect } from "../utils/types";
import { shapeRelationshipData } from "./shapeRelationshipData";
import { RelationshipType } from "./types";

type OrMany<T, IsOneToOne extends boolean> = IsOneToOne extends true ? T : T[];
type OrBoolean<T, IsOneToOne extends boolean> = IsOneToOne extends true ? boolean : T[];

// NOTE: The typing here is very relaxed, and really only validates that the Prisma 
// mutation structure is correct.
// Since yup validation and casting happens earlier, we don't need to 
// worry about type-specific validation here.
export type ShapeHelperOutput<
    IsOneToOne extends boolean,
    PrimaryKey extends string,
// > = {
//     connect: OrMany<RelConnect<PrimaryKey>, IsOneToOne>,
//     disconnect: OrBoolean<RelDisconnect<PrimaryKey>, IsOneToOne>,
//     delete: OrBoolean<RelDelete<PrimaryKey>, IsOneToOne>,
//     create: OrMany<RelCreate<any>, IsOneToOne>,
//     update: OrMany<RelUpdate<any, PrimaryKey>, IsOneToOne>,
// }
> = any; // TODO When set as "any", server starts. I think Prisma v5 changed its query shape

export type ShapeHelperProps<
    IsOneToOne extends boolean,
    Types extends readonly RelationshipType[],
    SoftDelete extends boolean = false,
> = {
    /** The data to convert */
    data: Record<string, any>,
    /** Ids which should result in a connect instead of a create */
    idsCreateToConnect?: IdsCreateToConnect,
    /**
     * True if relationship is one-to-one. This makes 
     * the results a single object instead of an array
     */
    isOneToOne: IsOneToOne,
    /**
    * If relationship is a join table, data required to create the join table record
    * 
    * NOTE: Does not differentiate between a disconnect and a delete. How these are handled is 
    * determined by the database cascading.
    */
    joinData?: {
        fieldName: string, // e.g. team.tags.tag => 'tag'
        uniqueFieldName: string, // e.g. team.tags.tag => 'team_tags_taggedid_tagTag_unique'
        childIdFieldName: string, // e.g. team.tags.tag => 'tagTag'
        parentIdFieldName: string, // e.g. team.tags.tag => 'taggedId'
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
    /** The allowed operations on the relations (e.g. create, connect) */
    relTypes: Types,
    /** Name of relation */
    relation: string,
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
 * @returns An object with the connections
 */
export const shapeHelper = async<
    IsOneToOne extends boolean,
    Types extends readonly RelationshipType[],
    PrimaryKey extends string = "id",
    SoftDelete extends boolean = false,
>({
    data,
    idsCreateToConnect = {},
    isOneToOne,
    joinData,
    objectType,
    parentRelationshipName,
    preMap,
    relation,
    relTypes,
    softDelete = false as SoftDelete,
    userData,
}: ShapeHelperProps<IsOneToOne, Types, SoftDelete>):
    Promise<ShapeHelperOutput<IsOneToOne, PrimaryKey>> => {
    // Initialize result
    let result: { [x: string]: any } = {};
    // Loop through relation types, and convert all to a Prisma-shaped array
    for (const t of relTypes) {
        // If not in data, skip
        const curr = data[`${relation}${t}` as string];
        if (!curr) continue;
        // Shape the data
        const currShaped = shapeRelationshipData(curr, [], false);
        // Add to result
        result[lowercaseFirstLetter(t)] = Array.isArray(result[lowercaseFirstLetter(t)]) ?
            [...result[lowercaseFirstLetter(t)], ...currShaped] :
            currShaped;
    }
    const { idField } = (ModelMap.getLogic(["idField"], objectType, false) ?? { idField: "id" }) as unknown as { idField: PrimaryKey };
    // Now we can further shape the result
    // Creates which show up in idsCreateToConnect should be converted to connects
    if (Array.isArray(result.create) && result.create.length > 0 && Object.keys(idsCreateToConnect).length > 0) {
        const connected = result.create.map((e: { [x: string]: any }) => {
            const id = idsCreateToConnect[e[idField]] ?? idsCreateToConnect[e.id];
            const isUuid = typeof id === "string" && uuidValidate(id);
            return id ? { [isUuid ? "id" : idField]: id } : null;
        }).filter((e) => e);
        if (connected.length) {
            result.connect = Array.isArray(result.connect) ? [...result.connect, ...connected] : connected;
            result.create = result.create.filter((e: { [x: string]: any }) => !idsCreateToConnect[e[idField]] && !idsCreateToConnect[e.id]);
        }
    }
    // Connects, diconnects, and deletes must be shaped in the form of { id: '123' } (i.e. no other fields)
    if (Array.isArray(result.connect) && result.connect.length > 0) {
        // Fallback to "id" if idField is not found
        result.connect = result.connect.map((e: { [x: string]: any }) => {
            if (e[idField]) return { [idField]: e[idField] };
            return { id: e.id };
        });
    } else result.connect = undefined;
    if (Array.isArray(result.disconnect) && result.disconnect.length > 0) {
        // Fallback to "id" if idField is not found
        result.disconnect = result.disconnect.map((e: { [x: string]: any }) => {
            if (e[idField]) return { [idField]: e[idField] };
            return { id: e.id };
        });
    } else result.disconnect = undefined;
    if (Array.isArray(result.delete) && result.delete.length > 0) {
        // Fallback to "id" if idField is not found
        result.delete = result.delete.map((e: { [x: string]: any }) => {
            if (e[idField]) return { [idField]: e[idField] };
            return { id: e.id };
        });
    } else result.delete = undefined;
    // Updates must be shaped in the form of { where: { id: '123' }, data: {...}}
    if (Array.isArray(result.update) && result.update.length > 0) {
        result.update = result.update.map((e: any) => ({ where: { id: e.id }, data: e }));
    }
    else result.update = undefined;
    // Convert deletes to updates if softDelete is true
    if (softDelete && Array.isArray(result.delete) && result.delete.length > 0) {
        const softDeletes = result.delete.map((e: any) => ({ where: { id: e[idField] }, data: { isDeleted: true } }));
        result.update = Array.isArray(result.update) ? [...result.update, ...softDeletes] : softDeletes;
        delete result.delete;
    }
    // Perform nested shapes for create and update
    const mutate = ModelMap.get(objectType, false)?.mutate;
    if (mutate?.shape.create && Array.isArray(result.create) && result.create.length > 0) {
        const shaped: { [x: string]: any }[] = [];
        for (const create of result.create) {
            const created = await mutate.shape.create({ data: create, idsCreateToConnect, preMap, userData });
            // Exclude parent relationship to prevent circular references
            const { [parentRelationshipName]: _, ...rest } = created;
            shaped.push(rest);
        }
        result.create = shaped;
    }
    if (mutate?.shape.update && Array.isArray(result.update) && result.update.length > 0) {
        const shaped: { [x: string]: any }[] = [];
        for (const update of result.update) {
            const updated = await mutate.shape.update({ data: update.data, idsCreateToConnect, preMap, userData });
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
            // delete: [ { team_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ] <-- A join table always deletes on disconnects
            for (const id of (result?.disconnect ?? [])) {
                const curr = {
                    [joinData.uniqueFieldName]: {
                        [joinData.childIdFieldName]: id[idField],
                        [joinData.parentIdFieldName]: joinData.parentId,
                    },
                };
                resultWithJoin.delete.push(curr);
            }
        }
        if (result.delete) {
            // delete: [ { team_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ]
            for (const id of (result?.delete ?? [])) {
                const curr = {
                    [joinData.uniqueFieldName]: {
                        [joinData.childIdFieldName]: id[idField],
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
            //         where: { team_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } },
            //         data: { tag: { update: { tag: 'fdas', } } }
            //     }]
            for (const data of (result?.update ?? [])) {
                const curr = {
                    where: { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: data.where[idField], [joinData.parentIdFieldName]: joinData.parentId } },
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
        // 1. Does not have both a connect and create
        // 2. Does not have both a disconnect and delete
        const isAdd = ("create" in result || "connect" in result) && !("delete" in result || "disconnect" in result || "update" in result);
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
    // If result is empty, return undefined
    // NOTE: To please the type checker, we pretend that we're returning a non-undefined value
    if (Object.keys(result).length === 0) return undefined as unknown as ShapeHelperOutput<IsOneToOne, PrimaryKey>;
    return result as ShapeHelperOutput<IsOneToOne, PrimaryKey>;
};

/** Typical relations for create inputs */
export const addRels = ["Create", "Connect"] as const;

/** Typical relations for update inputs */
export const updateRels = ["Connect", "Create", "Delete", "Disconnect", "Update"] as const;
