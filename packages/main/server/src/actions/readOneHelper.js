import { ViewFor } from "@local/consts";
import { getUser } from "../auth";
import { addSupplementalFields, modelToGql, selectHelper, toPartialGqlInfo } from "../builders";
import { CustomError } from "../events";
import { getIdFromHandle, getLatestVersion } from "../getters";
import { ObjectMap } from "../models";
import { ViewModel } from "../models/view";
import { getAuthenticatedData } from "../utils";
import { permissionsCheck } from "../validators";
export async function readOneHelper({ info, input, objectType, prisma, req, }) {
    const userData = getUser(req);
    const model = ObjectMap[objectType];
    if (!model)
        throw new CustomError("0350", "InternalError", req.languages, { objectType });
    if (!input.id && !input.idRoot && !input.handle && !input.handleRoot)
        throw new CustomError("0019", "IdOrHandleRequired", userData?.languages ?? req.languages);
    const partialInfo = toPartialGqlInfo(info, model.format.gqlRelMap, req.languages, true);
    let id;
    if (input.idRoot || input.handleRoot) {
        id = await getLatestVersion({ objectType: objectType, prisma, idRoot: input.idRoot, handleRoot: input.handleRoot });
    }
    else if (input.handle) {
        id = await getIdFromHandle({ handle: input.handle, objectType, prisma });
    }
    else {
        id = input.id;
    }
    if (!id)
        throw new CustomError("0434", "NotFound", userData?.languages ?? req.languages, { objectType });
    const authDataById = await getAuthenticatedData({ [model.__typename]: [id] }, prisma, userData ?? null);
    await permissionsCheck(authDataById, { ["Read"]: [id] }, userData);
    let object;
    try {
        object = await model.delegate(prisma).findUnique({ where: { id }, ...selectHelper(partialInfo) });
        if (!object)
            throw new CustomError("0022", "NotFound", userData?.languages ?? req.languages, { objectType });
    }
    catch (error) {
        throw new CustomError("0435", "NotFound", userData?.languages ?? req.languages, { objectType, error });
    }
    const formatted = modelToGql(object, partialInfo);
    if (userData?.id && objectType in ViewFor) {
        ViewModel.view(prisma, userData, { forId: object.id, viewFor: objectType });
    }
    const result = (await addSupplementalFields(prisma, userData, [formatted], partialInfo))[0];
    return result;
}
//# sourceMappingURL=readOneHelper.js.map