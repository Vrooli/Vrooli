import { assertRequestFrom } from "../auth/request";
import { CustomError } from "../events";
import { cudHelper } from "./cudHelper";
export async function deleteManyHelper({ input, objectType, prisma, req, }) {
    const userData = assertRequestFrom(req, { isUser: true });
    const { deleted } = await cudHelper({ deleteMany: input.ids, objectType, partialInfo: {}, prisma, userData });
    if (!deleted)
        throw new CustomError("0037", "InternalError", userData.languages);
    return deleted;
}
//# sourceMappingURL=deleteManyHelper.js.map