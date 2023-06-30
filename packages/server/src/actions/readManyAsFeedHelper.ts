import { modelToGql, toPartialGqlInfo } from "../builders";
import { getLogic } from "../getters";
import { readManyHelper } from "./readManyHelper";
import { ReadManyHelperProps } from "./types";

/**
 * Helper function for reading many objects and converting them to a GraphQL response
 * (except for supplemental fields). This is useful when querying feeds
 */
export async function readManyAsFeedHelper<Input extends { [x: string]: any }>({
    additionalQueries,
    info,
    input,
    objectType,
    prisma,
    req,
}: Omit<ReadManyHelperProps<Input>, "addSupplemental">): Promise<{ pageInfo: any, nodes: any[] }> {
    const readManyResult = await readManyHelper({
        additionalQueries,
        addSupplemental: false,
        info,
        input,
        objectType,
        prisma,
        req,
    });
    const { format } = getLogic(["format"], objectType, req.session.languages, "readManyAsFeedHelper");
    const nodes = readManyResult.edges.map(({ node }: any) =>
        modelToGql(node, toPartialGqlInfo(info, format.gqlRelMap, req.session.languages, true))) as any[];
    return {
        pageInfo: readManyResult.pageInfo,
        nodes,
    };
}
