import { VisibilityType } from "@local/consts";
import { getUser } from "../auth";
import { combineQueries, timeFrameToPrisma, visibilityBuilder } from "../builders";
import { getLogic } from "../getters";
export async function countHelper({ input, objectType, prisma, req, where, visibility = VisibilityType.Public, }) {
    const userData = getUser(req);
    const createdQuery = timeFrameToPrisma("created_at", input.createdTimeFrame);
    const updatedQuery = timeFrameToPrisma("updated_at", input.updatedTimeFrame);
    const visibilityQuery = visibilityBuilder({ objectType, userData, visibility });
    const { delegate } = getLogic(["delegate"], objectType, userData?.languages ?? ["en"], "countHelper");
    return await delegate(prisma).count({
        where: combineQueries([where, createdQuery, updatedQuery, visibilityQuery]),
    });
}
//# sourceMappingURL=countHelper.js.map