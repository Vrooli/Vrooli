import { createOwner } from "./createOwner";

/**
 * Shapes ownership connect fields for a GraphQL update input
 * @param item The item to shape, with an "owner" field
 * @returns Ownership connect object
 */
export const updateOwner = <
    Item extends { owner?: { type: 'Organization' | 'User', id: string } | null | undefined }
>(
    originalItem: Item,
    updatedItem: Item,
): {
    organizationConnect?: string | null | undefined;
    userConnect?: string | null | undefined;
} => {
    // Find owner data in item
    const originalOwnerData = originalItem.owner;
    const updatedOwnerData = updatedItem.owner;
    // If original and updated missing, return empty object
    if (originalOwnerData === null && originalOwnerData === undefined && updatedOwnerData === null && updatedOwnerData === undefined) return {};
    // If original and updated match, return empty object
    if (originalOwnerData?.id === updatedOwnerData?.id) return {};
    // If updated missing, disconnect
    // TODO disabled for now
    // Treat as create
    return createOwner(updatedItem);
};