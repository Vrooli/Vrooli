import { type expect as vitestExpect } from "vitest";

/**
 * Represents the common structure of a findMany result with edges and nodes containing an ID.
 */
type FindManyResultWithId = {
    edges?: ({ node?: { id?: bigint | string } | null | undefined } | null | undefined)[] | null | undefined;
} | null | undefined;

/**
 * Asserts that the IDs returned in a findMany result match the expected IDs.
 *
 * @param expect - The Vitest expect function.
 * @param result - The result object from a findMany operation.
 * @param expectedIds - An array of expected BigInt IDs.
 */
export function assertFindManyResultIds(
    expect: typeof vitestExpect,
    result: FindManyResultWithId,
    expectedIds: bigint[],
): void {
    expect(result).not.toBeNull();
    expect(result).toBeDefined();
    expect(result).toHaveProperty("edges");
    expect(result!.edges).toBeInstanceOf(Array);

    // Safely extract and convert result IDs to strings
    const resultIds = result!.edges!
        .map(edge => edge?.node?.id?.toString()) // Safely access and convert ID
        .filter((id): id is string => id !== undefined && id !== null); // Filter out undefined/null and assert type

    // Convert expected BigInt IDs to strings
    const expectedStringIds = expectedIds.map(id => id.toString());

    // Compare sorted arrays
    expect(resultIds.sort()).toEqual(expectedStringIds.sort());
} 
