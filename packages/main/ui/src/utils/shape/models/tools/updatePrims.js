import { DUMMY_ID, uuid } from "@local/uuid";
export const updatePrims = (original, updated, primary, ...fields) => {
    if (!updated || !original)
        return fields.reduce((acc, field) => ({ ...acc, [field]: undefined }), {});
    const changedFields = fields.reduce((acc, field) => ({ ...acc, [field]: updated[field] !== original[field] ? updated[field] : undefined }), {});
    if (!primary)
        return changedFields;
    if (primary !== "id")
        return { ...changedFields, [primary]: original[primary] };
    return { ...changedFields, [primary]: original[primary] === DUMMY_ID ? uuid() : original[primary] };
};
//# sourceMappingURL=updatePrims.js.map