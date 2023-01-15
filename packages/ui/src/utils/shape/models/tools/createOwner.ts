/**
 * Shapes ownership connect fields for a GraphQL create input
 * @param item The item to shape, with an "owner" field
 * @returns Ownership connect object
 */
export const createOwner = <
    Item extends { owner?: { __typename: 'User' | 'Organization', id: string } | null | undefined }
>(
    item: Item,
): {
    organizationConnect?: string | null | undefined;
    userConnect?: string | null | undefined;
} => {
    // Find owner data in item
    const ownerData = item.owner;
    // If owner data is undefined, or type is not a User or Organization return empty object
    if (ownerData === null || ownerData === undefined || (ownerData.__typename !== 'User' && ownerData.__typename !== 'Organization')) return {};
    // Initialize result
    const result: { [x: string]: any } = {};
    // Add connect field
    result[`${ownerData.__typename.toLowerCase()}Connect`] = ownerData.id;
    // Return result
    return result;
};