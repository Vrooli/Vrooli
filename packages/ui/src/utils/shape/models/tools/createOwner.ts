type OwnerPrefix = '' | 'ownedBy';
type OwnerType = 'User' | 'Organization';

/**
 * Shapes ownership connect fields for a GraphQL create input. Will only 
 * return 0 or 1 fields - cannot have two owners.
 * @param item The item to shape, with an owner field starting with the specified prefix
 * @returns Ownership connect object
 */
export const createOwner = <
    OType extends OwnerType,
    Item extends { owner?: { __typename: OType, id: string } | null | undefined },
    Prefix extends OwnerPrefix & string
>(
    item: Item,
    prefix: Prefix = '' as Prefix,
): { [K in `${Prefix}${OType}Connect`]?: string } => {
    // Find owner data in item
    const ownerData = item.owner;
    // If owner data is undefined, or type is not a User or Organization return empty object
    if (ownerData === null || ownerData === undefined || (ownerData.__typename !== 'User' && ownerData.__typename !== 'Organization')) return {};
    // Create field name (with first letter lowercase)
    let fieldName = `${prefix}${ownerData.__typename}Connect`;
    fieldName = fieldName.charAt(0).toLowerCase() + fieldName.slice(1);
    // Return shaped field
    return { [fieldName]: ownerData.id } as any;
};