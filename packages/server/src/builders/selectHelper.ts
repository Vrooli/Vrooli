import { isObject } from "@local/shared";
import { removeTypenames } from "./removeTypenames";
import { selPad } from "./selPad";
import { toPartialPrismaSelect } from "./toPartialPrismaSelect";
import { PartialGraphQLInfo, PartialPrismaSelect, PrismaSelect } from "./types";

/**
 * Converts shapes 2 and 3 of the GraphQL to Prisma conversion to shape 4.
 * @returns Object which can be passed into Prisma select directly
 */
export const selectHelper = (partial: PartialGraphQLInfo | PartialPrismaSelect): PrismaSelect | undefined => {
    // Convert partial's special cases (virtual/calculated fields, unions, etc.)
    let modified: { [x: string]: any } = toPartialPrismaSelect(partial);
    if (!isObject(modified)) return undefined;
    // Delete type fields
    modified = removeTypenames(modified);
    // Pad every relationship with "select"
    modified = selPad(modified);
    return modified as PrismaSelect;
};
