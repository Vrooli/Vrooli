import { difference } from "@shared/utils";
import { CustomError } from "../events";
import { linkToVersion } from "./linkToVersion";
import { shapeRelationshipData } from "./shapeRelationshipData";
import { BuiltRelationship, RelationshipBuilderHelperArgs, RelationshipTypes } from "./types";

/**
 * Converts an add or update's data to proper Prisma format. 
 * 
 * NOTE1: Must authenticate before calling this function!
 * 
 * NOTE2: Only goes one layer deep. You must handle grandchildren, great-grandchildren, etc. yourself
 */
export const relationshipBuilderHelper = async<
    IDField extends string,
    IsAdd extends boolean,
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    RelName extends string,
    Input extends { [x: string]: any },
    Shaped extends { [x: string]: any },
>({
    data,
    isAdd,
    fieldExcludes = [],
    idField = 'id' as IDField,
    isOneToOne = false as IsOneToOne,
    isRequired = false as IsRequired,
    isTransferable = true,
    linkVersion = false,
    joinData,
    prisma,
    relationshipName,
    relExcludes = [],
    shape,
    softDelete = false,
    userData,
}: RelationshipBuilderHelperArgs<IDField, IsAdd, IsOneToOne, IsRequired, RelName, Input>
): Promise<BuiltRelationship<IDField, IsAdd, IsOneToOne, IsRequired, Shaped>> => {
    // Determine valid operations, and remove operations that should be excluded
    let ops: RelationshipTypes[] = isAdd ? ['Connect', 'Create'] : ['Connect', 'Create', 'Disconnect', 'Delete', 'Update'];
    if (!isTransferable) ops = difference(ops, ['Connect', 'Disconnect']);
    ops = difference(ops, relExcludes);
    // Create result object
    let converted: { [x: string]: any } = {};
    // Loop through object's keys
    for (const [key, value] of Object.entries(data)) {
        if (value === null || value === undefined) continue;
        // Skip if not matching relationship or not a valid operation
        if (!key.startsWith(relationshipName) || !ops.some(o => key.toLowerCase().endsWith(o))) continue;
        // Determine operation. Versioned data may also have "Version" before "Connect", "Disconnect", and "Delete",
        // so we must remove that too
        const currOp = key.replace(relationshipName, '').replace('Version', '').toLowerCase();
        // TODO handle soft delete
        // Add operation to result object
        const shapedData = shapeRelationshipData(value, fieldExcludes, isOneToOne);
        // Should be an array if not one-to-one
        if (!isOneToOne) {
            converted[currOp] = Array.isArray(converted[currOp]) ? [...converted[currOp], ...shapedData] : shapedData;
        } else {
            converted[currOp] = shapedData;
        }
    };
    // Connects, diconnects, and deletes must be shaped in the form of { id: '123' } (i.e. no other fields)
    if (Array.isArray(converted.connect) && converted.connect.length > 0) converted.connect = converted.connect.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    if (Array.isArray(converted.disconnect) && converted.disconnect.length > 0) converted.disconnect = converted.disconnect.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    if (Array.isArray(converted.delete) && converted.delete.length > 0) converted.delete = converted.delete.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    // Updates must be shaped in the form of { where: { id: '123' }, data: {...}}
    if (Array.isArray(converted.update) && converted.update.length > 0) {
        converted.update = converted.update.map((e: any) => ({ where: { id: e.id }, data: e }));
    }
    // Shape creates and updates
    if (shape?.relCreate) {
        if (Array.isArray(converted.create)) {
            const shaped: { [x: string]: any }[] = [];
            for (const create of converted.create) {
                const created = await shape.relCreate({ data: create, prisma, userData });
                // If linkVersion is true, the data from relCreate must be flipped so 
                // version is top-level, and the rest is nested under "root"
                if (linkVersion) {
                    shaped.push(linkToVersion(created, true, userData.languages));
                } else {
                    shaped.push(created);
                }
            }
            converted.create = shaped;
        }
    }
    if (shape?.relUpdate) {
        if (Array.isArray(converted.update)) {
            const shaped: { where: { [key in IDField]: string }, data: { [x: string]: any } }[] = [];
            for (const update of converted.update) {
                const updated = await shape.relUpdate({ data: update.data, prisma, userData, where: update.where });
                // If linkVersion is true, the data from relUpdate must be flipped so 
                // version is top-level, and the rest is nested under "root"
                if (linkVersion) {
                    shaped.push({ where: update.where, data: linkToVersion(updated, false, userData.languages) });
                } else {
                    shaped.push({ where: update.where, data: updated });
                }
            }
            converted.update = shaped;
        }
    }
    // Handle join table, if applicable
    if (joinData) {
        if (converted.connect) {
            // ex: create: [ { tag: { connect: { id: 'asdf' } } } ] <-- A join table always creates on connects
            for (const id of (converted?.connect ?? [])) {
                const curr = { [joinData.fieldName]: { connect: id } };
                converted.create = Array.isArray(converted.create) ? [...converted.create, curr] : [curr];
            }
        }
        if (converted.disconnect) {
            // delete: [ { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ] <-- A join table always deletes on disconnects
            for (const id of (converted?.disconnect ?? [])) {
                const curr = { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: id[idField], [joinData.parentIdFieldName]: joinData.parentId } };
                converted.delete = Array.isArray(converted.delete) ? [...converted.delete, curr] : [curr];
            }
        }
        if (converted.delete) {
            // delete: [ { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ]
            for (const id of (converted?.delete ?? [])) {
                const curr = { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: id[idField], [joinData.parentIdFieldName]: joinData.parentId } };
                converted.delete = Array.isArray(converted.delete) ? [...converted.delete, curr] : [curr];
            }
        }
        if (converted.create) {
            // ex: create: [ { tag: { create: { id: 'asdf' } } } ]
            for (const id of (converted?.create ?? [])) {
                const curr = { [joinData.fieldName]: { create: id } };
                converted.create = Array.isArray(converted.create) ? [...converted.create, curr] : [curr];
            }
        }
        if (converted.update) {
            // ex: update: [{ 
            //         where: { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } },
            //         data: { tag: { update: { tag: 'fdas', } } }
            //     }]
            for (const data of (converted?.update ?? [])) {
                const curr = {
                    where: { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: data.where[idField], [joinData.parentIdFieldName]: joinData.parentId } },
                    data: { [joinData.fieldName]: { update: data.data } }
                };
                converted.update = Array.isArray(converted.update) ? [...converted.update, curr] : [curr];
            }
        }
    }
    // If isOneToOne, perform the following checks:
    // 1. If required:
    //    1.a If adding, must have connect or create
    //    a.b If updating and has a disconnect or delete, must have connect or create
    // 3. Does not have both a connect and create
    // 4. Does not have both a disconnect and delete
    if (isOneToOne) {
        if (isRequired) {
            if (isAdd && !converted.connect && !converted.create)
                throw new CustomError('0340', 'InvalidArgs', userData.languages, { relationshipName });
            if (!isAdd && (converted.disconnect || converted.delete) && !converted.connect && !converted.create)
                throw new CustomError('0341', 'InvalidArgs', userData.languages, { relationshipName });
        }
        if (converted.connect && converted.create)
            throw new CustomError('0342', 'InvalidArgs', userData.languages, { relationshipName });
        if (converted.disconnect && converted.delete)
            throw new CustomError('0343', 'InvalidArgs', userData.languages, { relationshipName });
    }
    // Convert to correct format, depending on whether it is a one-to-one or one-to-many
    if (isOneToOne) {
        // one-to-one's disconnect/delete must be true or undefined
        if (converted.disconnect) converted.disconnect = true;
        if (converted.delete) converted.delete = true;
        // one-to-one's connect/create must be an object, not an array
        if (Array.isArray(converted.connect)) converted.connect = converted.connect.length ? converted.connect[0] : undefined;
        if (Array.isArray(converted.create)) converted.create = converted.create.length ? converted.create[0] : undefined;
        // one-to-one's update must not have a where, and must be an object, not an array
        if (Array.isArray(converted.update)) converted.update = converted.update.length ? converted.update[0].data : undefined;
    }
    // If one-to-many, should already be in correct format
    return converted;
}