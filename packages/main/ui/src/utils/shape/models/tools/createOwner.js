export const createOwner = (item, prefix = "") => {
    const ownerData = item.owner;
    if (ownerData === null || ownerData === undefined || (ownerData.__typename !== "User" && ownerData.__typename !== "Organization"))
        return {};
    let fieldName = `${prefix}${ownerData.__typename}Connect`;
    fieldName = fieldName.charAt(0).toLowerCase() + fieldName.slice(1);
    return { [fieldName]: ownerData.id };
};
//# sourceMappingURL=createOwner.js.map