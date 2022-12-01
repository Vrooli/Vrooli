import { modelToGraphQL, toPartialGraphQLInfo } from "../builders"
import { getFormatter } from "../getters"
import { readManyHelper } from "./readManyHelper"
import { ReadManyHelperProps } from "./types"

/**
 * Helper function for reading many objects and converting them to a GraphQL response
 * (except for supplemental fields). This is useful when querying feeds
 */
 export async function readManyAsFeedHelper<GraphQLModel extends { [x: string]: any }>({
    additionalQueries,
    info,
    input,
    objectType,
    prisma,
    req,
}: Omit<ReadManyHelperProps<GraphQLModel>, 'addSupplemental'>): Promise<{ pageInfo: any, nodes: any[] }> {
    const readManyResult = await readManyHelper({
        additionalQueries,
        addSupplemental: false,
        info,
        input,
        objectType,
        prisma,
        req,
    })
    const format = getFormatter(objectType, req.languages, 'readManyAsFeedHelper')
    const nodes = readManyResult.edges.map(({ node }: any) =>
        modelToGraphQL(node, toPartialGraphQLInfo(info, format.relationshipMap, req.languages, true))) as any[]
    return {
        pageInfo: readManyResult.pageInfo,
        nodes,
    }
}