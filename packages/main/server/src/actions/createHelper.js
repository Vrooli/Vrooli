import { assertRequestFrom } from "../auth/request";
import { addSupplementalFields, toPartialGqlInfo } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { cudHelper } from "./cudHelper";
export async function createHelper({ info, input, objectType, prisma, req, }) {
    const userData = assertRequestFrom(req, { isUser: true });
    const { format } = getLogic(["format"], objectType, req.languages, "createHelper");
    const partialInfo = toPartialGqlInfo(info, format.gqlRelMap, req.languages, true);
    const cudResult = await cudHelper({ createMany: [input], objectType, partialInfo, prisma, userData });
    const { created } = cudResult;
    if (created && created.length > 0) {
        return (await addSupplementalFields(prisma, userData, created, partialInfo))[0];
    }
    throw new CustomError("0028", "ErrorUnknown", userData.languages);
}
//# sourceMappingURL=createHelper.js.map