import { modelToGraphQL, toPartialGraphQLInfo } from "../builders"
import { getLogic } from "../getters"
import { readManyHelper } from "./readManyHelper"
import { ReadManyHelperProps } from "./types"

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
}: Omit<ReadManyHelperProps<Input>, 'addSupplemental'>): Promise<{ pageInfo: any, nodes: any[] }> {
    console.log('readmanyasfeedhelper start', JSON.stringify(additionalQueries), JSON.stringify(input), '\n');
    const readManyResult = await readManyHelper({
        additionalQueries,
        addSupplemental: false,
        info,
        input,
        objectType,
        prisma,
        req,
    })
    const { format } = getLogic(['format'], objectType, req.languages, 'readManyAsFeedHelper')
    const nodes = readManyResult.edges.map(({ node }: any) =>
        modelToGraphQL(node, toPartialGraphQLInfo(info, format.gqlRelMap, req.languages, true))) as any[]
    return {
        pageInfo: readManyResult.pageInfo,
        nodes,
    }
}