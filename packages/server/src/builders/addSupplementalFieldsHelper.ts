import { ObjectMap } from "../models";
import { GraphQLModelType, SupplementalConverter } from "../models/types";
import { SessionUser } from "../endpoints/types";
import { PrismaType, RecursivePartial } from "../types";
import { PartialGraphQLInfo } from "./types";

/**
 * Adds supplemental fields data to the given objects
 */
export const addSupplementalFieldsHelper = async <GraphQLModel extends { [x: string]: any }>({ languages, objects, objectType, partial, prisma, userData }: {
    languages: string[],
    objects: ({ id: string } & { [x: string]: any })[],
    objectType: GraphQLModelType,
    partial: PartialGraphQLInfo,
    prisma: PrismaType,
    userData: SessionUser | null,
}): Promise<RecursivePartial<GraphQLModel>[]> => {
    if (!objects || objects.length === 0) return [];
    // Get supplemental info for object
    const supplementer: SupplementalConverter<any> | undefined = ObjectMap[objectType]?.format?.supplemental;
    if (!supplementer) return objects;
    // Get IDs from objects
    const ids = objects.map(({ id }) => id);
    // Call each resolver to get supplemental data
    const resolvers = supplementer.toGraphQL({ ids, languages, objects, partial, prisma, userData });
    for (const [field, resolver] of Object.entries(resolvers)) {
        // If not in partial, skip
        if (!partial[field]) continue;
        const supplemental = await resolver();
        objects = objects.map((x, i) => ({ ...x, [field]: supplemental[i] }));
    }
    return objects;
}