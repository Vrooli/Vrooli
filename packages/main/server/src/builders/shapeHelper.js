import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { shapeRelationshipData } from "./shapeRelationshipData";
export const shapeHelper = async ({ data, isOneToOne, isRequired, joinData, objectType, parentRelationshipName, preMap, primaryKey = "id", prisma, relation, relTypes, softDelete = false, userData, }) => {
    const result = {};
    if (!data[`${relation}Connect`] && !data[`${relation}Create`] && isRequired) {
        throw new CustomError("0368", "InvalidArgs", ["en"], { relation, data });
    }
    for (const t of relTypes) {
        const curr = data[`${relation}${t}`];
        if (!curr)
            continue;
        const currShaped = shapeRelationshipData(curr, [parentRelationshipName, `${parentRelationshipName}Id`]);
        result[t.toLowerCase()] = Array.isArray(result[t.toLowerCase()]) ?
            [...result[t.toLowerCase()], ...currShaped] :
            currShaped;
    }
    if (Array.isArray(result.connect) && result.connect.length > 0)
        result.connect = result.connect.map((e) => ({ [primaryKey]: e[primaryKey] }));
    if (Array.isArray(result.disconnect) && result.disconnect.length > 0)
        result.disconnect = result.disconnect.map((e) => ({ [primaryKey]: e[primaryKey] }));
    if (Array.isArray(result.delete) && result.delete.length > 0)
        result.delete = result.delete.map((e) => ({ [primaryKey]: e[primaryKey] }));
    if (Array.isArray(result.update) && result.update.length > 0) {
        result.update = result.update.map((e) => ({ where: { id: e.id }, data: e }));
    }
    if (softDelete && Array.isArray(result.delete) && result.delete.length > 0) {
        const softDeletes = result.delete.map((e) => ({ where: { id: e[primaryKey] }, data: { isDeleted: true } }));
        result.update = Array.isArray(result.update) ? [...result.update, ...softDeletes] : softDeletes;
        delete result.delete;
    }
    const mutate = ObjectMap[objectType]?.mutate;
    if (mutate?.shape.create && Array.isArray(result.create) && result.create.length > 0) {
        const shaped = [];
        for (const create of result.create) {
            const created = await mutate.shape.create({ data: create, preMap, prisma, userData });
            shaped.push(created);
        }
        result.create = shaped;
    }
    if (mutate?.shape.update && Array.isArray(result.update) && result.update.length > 0) {
        const shaped = [];
        for (const update of result.update) {
            const updated = await mutate.shape.update({ data: update, preMap, prisma, userData });
            shaped.push({ where: update.where, data: updated });
        }
        result.update = shaped;
    }
    if (joinData) {
        if (result.connect) {
            for (const id of (result?.connect ?? [])) {
                const curr = { [joinData.fieldName]: { connect: id } };
                result.create = Array.isArray(result.create) ? [...result.create, curr] : [curr];
            }
        }
        if (result.disconnect) {
            for (const id of (result?.disconnect ?? [])) {
                const curr = { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: id[primaryKey], [joinData.parentIdFieldName]: joinData.parentId } };
                result.delete = Array.isArray(result.delete) ? [...result.delete, curr] : [curr];
            }
        }
        if (result.delete) {
            for (const id of (result?.delete ?? [])) {
                const curr = { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: id[primaryKey], [joinData.parentIdFieldName]: joinData.parentId } };
                result.delete = Array.isArray(result.delete) ? [...result.delete, curr] : [curr];
            }
        }
        if (result.create) {
            for (const id of (result?.create ?? [])) {
                const curr = { [joinData.fieldName]: { create: id } };
                result.create = Array.isArray(result.create) ? [...result.create, curr] : [curr];
            }
        }
        if (result.update) {
            for (const data of (result?.update ?? [])) {
                const curr = {
                    where: { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: data.where[primaryKey], [joinData.parentIdFieldName]: joinData.parentId } },
                    data: { [joinData.fieldName]: { update: data.data } },
                };
                result.update = Array.isArray(result.update) ? [...result.update, curr] : [curr];
            }
        }
    }
    if (isOneToOne) {
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
        if (result.disconnect)
            result.disconnect = true;
        if (result.delete)
            result.delete = true;
        if (Array.isArray(result.connect))
            result.connect = result.connect.length ? result.connect[0] : undefined;
        if (Array.isArray(result.create))
            result.create = result.create.length ? result.create[0] : undefined;
        if (Array.isArray(result.update))
            result.update = result.update.length ? result.update[0].data : undefined;
    }
    if (!Object.keys(result).length)
        return {};
    return { [relation]: result };
};
export const addRels = ["Create", "Connect"];
export const updateRels = ["Connect", "Create", "Delete", "Disconnect", "Update"];
//# sourceMappingURL=shapeHelper.js.map