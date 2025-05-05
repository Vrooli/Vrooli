import { expect as chaiExpect } from "chai";

/**
 * Represents the common structure of a findMany result with edges and nodes containing an ID.
 */
type FindManyResultWithId = {
    edges?: ({ node?: { id?: bigint | string } | null | undefined } | null | undefined)[] | null | undefined;
} | null | undefined;

/**
 * Asserts that the IDs returned in a findMany result match the expected IDs.
 *
 * @param expect - The Chai expect function.
 * @param result - The result object from a findMany operation.
 * @param expectedIds - An array of expected BigInt IDs.
 */
export function assertFindManyResultIds(
    expect: typeof chaiExpect,
    result: FindManyResultWithId,
    expectedIds: bigint[],
): void {
    expect(result).to.not.be.null;
    expect(result).to.not.be.undefined;
    expect(result).to.have.property("edges").that.is.an("array");

    // Safely extract and convert result IDs to strings
    const resultIds = result!.edges!
        .map(edge => edge?.node?.id?.toString()) // Safely access and convert ID
        .filter((id): id is string => id !== undefined && id !== null); // Filter out undefined/null and assert type

    // Convert expected BigInt IDs to strings
    const expectedStringIds = expectedIds.map(id => id.toString());

    // Compare sorted arrays
    expect(resultIds.sort()).to.deep.equal(expectedStringIds.sort());
} 
