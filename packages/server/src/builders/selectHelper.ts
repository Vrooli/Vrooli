import { isObject } from "@shared/utils";
import { padSelect } from "./padSelect";
import { removeTypenames } from "./removeTypenames";
import { toPartialPrismaSelect } from "./toPartialPrismaSelect";
import { PartialGraphQLInfo, PartialPrismaSelect, PrismaSelect } from "./types";

/**
 * Converts shapes 2 and 3 of the GraphQL to Prisma conversion to shape 4.
 * @returns Object which can be passed into Prisma select directly
 */
export const selectHelper = (partial: PartialGraphQLInfo | PartialPrismaSelect): PrismaSelect | undefined => {
    console.log('selecthelper starttt', JSON.stringify(partial))
    // Convert partial's special cases (virtual/calculated fields, unions, etc.)
    let modified: { [x: string]: any } = toPartialPrismaSelect(partial);
    if (!isObject(modified)) return undefined;
    // Delete type fields
    modified = removeTypenames(modified);
    console.log('selecthelper after removeTypenames', JSON.stringify(modified));
    // Pad every relationship with "select"
    console.log('selecthelper before padSelect', JSON.stringify(modified));
    modified = padSelect(modified);
    console.log('selectHelper end', JSON.stringify(modified), '\n\n');
    return modified as PrismaSelect;
}