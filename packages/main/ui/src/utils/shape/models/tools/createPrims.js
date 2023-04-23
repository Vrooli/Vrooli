import { DUMMY_ID, uuid } from "@local/uuid";
export const createPrims = (object, ...fields) => {
    const prims = fields.reduce((acc, field) => ({ ...acc, [field]: object[field] !== null ? object[field] : undefined }), {});
    if (!fields.includes("id"))
        return prims;
    return { ...prims, id: prims.id === DUMMY_ID ? uuid() : prims.id };
};
//# sourceMappingURL=createPrims.js.map