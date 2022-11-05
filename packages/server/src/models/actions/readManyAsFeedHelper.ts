import { modelToGraphQL, toPartialGraphQLInfo } from "../builder"
import { SearchInputBase, PartialGraphQLInfo } from "../types"
import { readManyHelper } from "./readManyHelper"
import { ReadManyHelperProps } from "./types"

/**
 * Helper function for reading many objects and converting them to a GraphQL response
 * (except for supplemental fields). This is useful when querying feeds
 */
 export async function readManyAsFeedHelper<GraphQLModel, SearchInput extends SearchInputBase<any>>({
    additionalQueries,
    info,
    input,
    model,
    prisma,
    req,
}: Omit<ReadManyHelperProps<GraphQLModel, SearchInput>, 'addSupplemental'>): Promise<{ pageInfo: any, nodes: any[] }> {
    const readManyResult = await readManyHelper({
        additionalQueries,
        addSupplemental: false,
        info,
        input,
        model,
        prisma,
        req,
    })
    const nodes = readManyResult.edges.map(({ node }: any) =>
        modelToGraphQL(node, toPartialGraphQLInfo(info, model.format.relationshipMap) as PartialGraphQLInfo)) as any[]
    return {
        pageInfo: readManyResult.pageInfo,
        nodes,
    }
}