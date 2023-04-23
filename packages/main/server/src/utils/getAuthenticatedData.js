import { permissionsSelectHelper } from "../builders";
import { getLogic } from "../getters";
export const getAuthenticatedData = async (idsByType, prisma, userData) => {
    const authDataById = {};
    for (const type of Object.keys(idsByType)) {
        const { delegate, validate } = getLogic(["delegate", "validate"], type, userData?.languages ?? ["en"], "getAuthenticatedData");
        const data = await delegate(prisma).findMany({
            where: { id: { in: idsByType[type] } },
            select: permissionsSelectHelper(validate.permissionsSelect, userData?.id ?? null, userData?.languages ?? ["en"]),
        });
        for (const datum of data) {
            authDataById[datum.id] = { __typename: type, ...datum };
        }
    }
    return authDataById;
};
//# sourceMappingURL=getAuthenticatedData.js.map