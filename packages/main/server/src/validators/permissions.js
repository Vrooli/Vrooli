import { permissionsSelectHelper } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { isOwnerAdminCheck } from "./isOwnerAdminCheck";
export async function getMultiTypePermissions(authDataById, userData) {
    const permissionsById = {};
    for (const id of Object.keys(authDataById)) {
        const { validate } = getLogic(["validate"], authDataById[id].__typename, userData?.languages ?? ["en"], "getMultiplePermissions");
        const isAdmin = userData?.id ? isOwnerAdminCheck(validate.owner(authDataById[id], userData.id), userData.id) : false;
        const isDeleted = validate.isDeleted(authDataById[id], userData?.languages ?? ["en"]);
        const isLoggedIn = !!userData?.id;
        const isPublic = validate.isPublic(authDataById[id], userData?.languages ?? ["en"]);
        const permissionResolvers = validate.permissionResolvers({ isAdmin, isDeleted, isLoggedIn, isPublic, data: authDataById[id] });
        const permissions = await Promise.all(Object.entries(permissionResolvers).map(async ([key, resolver]) => [key, await resolver()])).then(entries => Object.fromEntries(entries));
        permissionsById[id] = permissions;
    }
    return permissionsById;
}
export async function getSingleTypePermissions(type, ids, prisma, userData) {
    const permissions = {};
    const { delegate, validate } = getLogic(["delegate", "validate"], type, userData?.languages ?? ["en"], "getSingleTypePermissions");
    let select;
    let authData = [];
    try {
        select = permissionsSelectHelper(validate.permissionsSelect, userData?.id ?? null, userData?.languages ?? ["en"]);
        authData = await delegate(prisma).findMany({
            where: { id: { in: ids } },
            select,
        });
    }
    catch (error) {
        throw new CustomError("0388", "InternalError", userData?.languages ?? ["en"], { ids, select, objectType: type });
    }
    for (const authDataItem of authData) {
        const isAdmin = userData?.id ? isOwnerAdminCheck(validate.owner(authDataItem, userData.id), userData.id) : false;
        const isDeleted = validate.isDeleted(authDataItem, userData?.languages ?? ["en"]);
        const isLoggedIn = !!userData?.id;
        const isPublic = validate.isPublic(authDataItem, userData?.languages ?? ["en"]);
        const permissionResolvers = validate.permissionResolvers({ isAdmin, isDeleted, isLoggedIn, isPublic, data: authDataItem });
        const permissionsObject = Object.fromEntries(Object.entries(permissionResolvers).map(([key, resolver]) => [key, resolver()]));
        for (const key of Object.keys(permissionsObject)) {
            permissions[key] = [...(permissions[key] ?? []), permissionsObject[key]];
        }
    }
    return permissions;
}
export async function permissionsCheck(authDataById, idsByAction, userData) {
    const permissionsById = await getMultiTypePermissions(authDataById, userData);
    for (const action of Object.keys(idsByAction)) {
        const ids = idsByAction[action];
        for (const id of ids) {
            const permissions = permissionsById[id];
            if (!permissions) {
                if (action !== "Create") {
                    throw new CustomError("0390", "InternalError", userData?.languages ?? ["en"], { action, id, __typename: authDataById[id].__typename });
                }
                continue;
            }
            if (`can${action}` in permissions && !permissions[`can${action}`]) {
                throw new CustomError("0297", "Unauthorized", userData?.languages ?? ["en"], { action, id, __typename: authDataById[id].__typename });
            }
        }
    }
}
//# sourceMappingURL=permissions.js.map