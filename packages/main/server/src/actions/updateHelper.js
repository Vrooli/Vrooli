import { assertRequestFrom } from "../auth/request";
import { addSupplementalFields, toPartialGqlInfo } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { cudHelper } from "./cudHelper";
export async function updateHelper({ info, input, objectType, prisma, req, where = (obj) => ({ id: obj.id }), }) {
    const userData = assertRequestFrom(req, { isUser: true });
    const { format } = getLogic(["format"], objectType, userData.languages, "cudHelper");
    const partialInfo = toPartialGqlInfo(info, format.gqlRelMap, req.languages, true);
    const shapedInput = { where: where(input), data: input };
    const cudResult = await cudHelper({ updateMany: [shapedInput], objectType, partialInfo, prisma, userData });
    const { updated } = cudResult;
    if (updated && updated.length > 0) {
        return (await addSupplementalFields(prisma, userData, updated, partialInfo))[0];
    }
    throw new CustomError("0032", "ErrorUnknown", userData.languages);
}
//# sourceMappingURL=updateHelper.js.map