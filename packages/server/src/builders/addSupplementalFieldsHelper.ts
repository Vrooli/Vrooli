import { ObjectMap } from "../models";
import { DotNotation, GqlModelType, SessionUser } from '@shared/consts';
import { PrismaType, RecursivePartial } from "../types";
import { PartialGraphQLInfo } from "./types";
import { SupplementalConverter } from "../models/types";
import { setDotNotationValue } from "@shared/utils";

/**
 * Adds supplemental fields data to the given objects
 */
export const addSupplementalFieldsHelper = async <GraphQLModel extends { [x: string]: any }>({ languages, objects, objectType, partial, prisma, userData }: {
    languages: string[],
    objects: ({ id: string } & { [x: string]: any })[],
    objectType: `${GqlModelType}`,
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
    // Get supplemental data by field
    const supplementalData = await supplementer.toGraphQL({ ids, languages, objects, partial, prisma, userData });
    // Loop through each field in supplemental data
    for (const [field, data] of Object.entries(supplementalData)) {
        // Each value is an array of data for each object, in the same order as the objects
        // Set the value for each object
        objects = objects.map((x, i) => setDotNotationValue(x, field as never, data[i])) as any[];
    }
    return objects;
}