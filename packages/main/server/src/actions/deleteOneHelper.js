import { assertRequestFrom } from "../auth/request";
import { cudHelper } from "./cudHelper";
export async function deleteOneHelper({ input, objectType, prisma, req, }) {
    const userData = assertRequestFrom(req, { isUser: true });
    const { deleted } = await cudHelper({ deleteMany: [input.id], objectType, partialInfo: {}, prisma, userData });
    return { __typename: "Success", success: Boolean(deleted?.count && deleted.count > 0) };
}
//# sourceMappingURL=deleteOneHelper.js.map