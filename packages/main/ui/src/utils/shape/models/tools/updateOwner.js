import { createOwner } from "./createOwner";
export const updateOwner = (originalItem, updatedItem, prefix = "") => {
    const originalOwnerData = originalItem.owner;
    const updatedOwnerData = updatedItem.owner;
    if ((originalOwnerData === null || originalOwnerData === undefined) &&
        (updatedOwnerData === null || updatedOwnerData === undefined))
        return {};
    if (originalOwnerData?.id === updatedOwnerData?.id)
        return {};
    return createOwner(updatedItem, prefix);
};
//# sourceMappingURL=updateOwner.js.map