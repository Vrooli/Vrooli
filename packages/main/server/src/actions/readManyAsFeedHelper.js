import { modelToGql, toPartialGqlInfo } from "../builders";
import { getLogic } from "../getters";
import { readManyHelper } from "./readManyHelper";
export async function readManyAsFeedHelper({ additionalQueries, info, input, objectType, prisma, req, }) {
    const readManyResult = await readManyHelper({
        additionalQueries,
        addSupplemental: false,
        info,
        input,
        objectType,
        prisma,
        req,
    });
    const { format } = getLogic(["format"], objectType, req.languages, "readManyAsFeedHelper");
    const nodes = readManyResult.edges.map(({ node }) => modelToGql(node, toPartialGqlInfo(info, format.gqlRelMap, req.languages, true)));
    return {
        pageInfo: readManyResult.pageInfo,
        nodes,
    };
}
//# sourceMappingURL=readManyAsFeedHelper.js.map