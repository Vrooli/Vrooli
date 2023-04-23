import { reqArr } from "@local/validation";
import { modelToGql, selectHelper } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { cudInputsToMaps, getAuthenticatedData } from "../utils";
import { maxObjectsCheck, permissionsCheck, profanityCheck } from "../validators";
export async function cudHelper({ createMany, deleteMany, objectType, partialInfo, prisma, updateMany, userData, }) {
    const { delegate, mutate, validate } = getLogic(["delegate", "mutate", "validate"], objectType, userData.languages, "cudHelper");
    let created = [], updated = [], deleted = { __typename: "Count", count: 0 };
    let createAuthData = {}, updateAuthData = {};
    createMany && mutate.yup.create && reqArr(mutate.yup.create({})).validateSync(createMany, { abortEarly: false });
    updateMany && mutate.yup.update && reqArr(mutate.yup.update({})).validateSync(updateMany.map(u => u.data), { abortEarly: false });
    createMany && profanityCheck(createMany, partialInfo.__typename, userData.languages);
    updateMany && profanityCheck(updateMany.map(u => u.data), partialInfo.__typename, userData.languages);
    const { idsByAction, idsByType, inputsByType } = await cudInputsToMaps({
        createMany,
        updateMany,
        deleteMany,
        objectType,
        prisma,
        languages: userData.languages,
    });
    const preMap = {};
    for (const [type, inputs] of Object.entries(inputsByType)) {
        preMap[type] = {};
        const { mutate } = getLogic(["mutate"], type, userData.languages, "preshape type");
        if (mutate.shape.pre) {
            const { Create: createList, Update: updateList, Delete: deleteList } = inputs;
            const preResult = await mutate.shape.pre({ createList, updateList: updateList, deleteList, prisma, userData });
            preMap[type] = preResult;
        }
    }
    const shapedCreate = [];
    const shapedUpdate = [];
    if (createMany && mutate.shape.create) {
        for (const create of createMany) {
            shapedCreate.push(await mutate.shape.create({ data: create, preMap, prisma, userData }));
        }
    }
    if (updateMany && mutate.shape.update) {
        for (const update of updateMany) {
            const shaped = await mutate.shape.update({ data: update.data, preMap, prisma, userData, where: update.where });
            shapedUpdate.push({ where: update.where, data: shaped });
        }
    }
    const authDataById = await getAuthenticatedData(idsByType, prisma, userData);
    await permissionsCheck(authDataById, idsByAction, userData);
    maxObjectsCheck(authDataById, idsByAction, prisma, userData);
    if (shapedCreate.length > 0) {
        for (const data of shapedCreate) {
            let createResult = {};
            let select;
            try {
                select = selectHelper(partialInfo)?.select;
                createResult = await delegate(prisma).create({
                    data,
                    select,
                });
            }
            catch (error) {
                throw new CustomError("0415", "InternalError", userData.languages, { error, data, select, objectType });
            }
            const converted = modelToGql(createResult, partialInfo);
            created.push(converted);
        }
        createAuthData = Object.fromEntries(Object.entries(authDataById).filter(([id]) => created.map(c => c.id).includes(id)));
        mutate.trigger?.onCreated && await mutate.trigger.onCreated({
            authData: createAuthData,
            created,
            preMap,
            prisma,
            userData,
        });
    }
    if (shapedUpdate.length > 0) {
        for (const update of shapedUpdate) {
            let updateResult = {};
            let select;
            try {
                select = selectHelper(partialInfo)?.select;
                updateResult = await delegate(prisma).update({
                    where: update.where,
                    data: update.data,
                    select,
                });
            }
            catch (error) {
                throw new CustomError("0416", "InternalError", userData.languages, { error, update, select, objectType });
            }
            const converted = modelToGql(updateResult, partialInfo);
            updated.push(converted);
        }
        updateAuthData = Object.fromEntries(Object.entries(authDataById).filter(([id]) => updated.map(u => u.id).includes(id)));
        mutate.trigger?.onUpdated && await mutate.trigger.onUpdated({
            authData: updateAuthData,
            preMap,
            prisma,
            updated,
            updateInput: updateMany.map(u => u.data), userData,
        });
    }
    if (deleteMany && deleteMany.length > 0) {
        let beforeDeletedData;
        if (mutate.trigger?.beforeDeleted) {
            beforeDeletedData = await mutate.trigger.beforeDeleted({ deletingIds: deleteMany, prisma, userData });
        }
        const where = { id: { in: deleteMany } };
        try {
            deleted = await delegate(prisma).deleteMany({
                where,
            }).then(({ count }) => ({ __typename: "Count", count }));
        }
        catch (error) {
            throw new CustomError("0417", "InternalError", userData.languages, { error, where, objectType });
        }
        mutate.trigger?.onDeleted && await mutate.trigger.onDeleted({
            beforeDeletedData,
            deleted,
            deletedIds: deleteMany,
            preMap,
            prisma,
            userData,
        });
    }
    if (shapedCreate.length > 0 || shapedUpdate.length > 0 || (deleteMany && deleteMany.length > 0)) {
        mutate.trigger?.onCommon && await mutate.trigger.onCommon({
            createAuthData,
            created,
            deleted,
            deletedIds: deleteMany ?? [],
            preMap,
            prisma,
            updateAuthData,
            updated,
            updateInput: updateMany?.map(u => u.data) ?? [],
            userData,
        });
    }
    for (const type in inputsByType) {
        const { mutate } = getLogic(["mutate"], type, userData.languages, "postshape type");
        if (mutate.shape.post) {
            await mutate.shape.post({
                created,
                deletedIds: deleteMany ?? [],
                prisma,
                updated,
                userData,
            });
        }
    }
    return {
        created: createMany ? created : undefined,
        updated: updateMany ? updated : undefined,
        deleted: deleteMany ? deleted : undefined,
    };
}
//# sourceMappingURL=cudHelper.js.map