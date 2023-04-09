import { createOwner } from "./createOwner";

type OwnerPrefix = '' | 'ownedBy';
type OwnerType = 'User' | 'Organization';

/**
 * Shapes ownership connect fields for a GraphQL update input
 * @param item The item to shape, with an "owner" field
 * @returns Ownership connect object
 */
export const updateOwner = <
    OType extends OwnerType,
    OriginalItem extends { owner?: { __typename: OwnerType, id: string } | null | undefined },
    UpdatedItem extends { owner?: { __typename: OType, id: string } | null | undefined },
    Prefix extends OwnerPrefix & string
>(
    originalItem: OriginalItem,
    updatedItem: UpdatedItem,
    prefix: Prefix = '' as Prefix,
): { [K in `${Prefix}${OType}Connect`]?: string } => {
    // Find owner data in item
    const originalOwnerData = originalItem.owner;
    const updatedOwnerData = updatedItem.owner;
    // If original and updated missing
    if (
        (originalOwnerData === null || originalOwnerData === undefined) &&
        (updatedOwnerData === null || updatedOwnerData === undefined)
    ) return {};
    // If original and updated match, return empty object
    if (originalOwnerData?.id === updatedOwnerData?.id) return {};
    // If updated missing, disconnect
    // TODO disabled for now
    // Treat as create
    return createOwner(updatedItem as any, prefix);
};