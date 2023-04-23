import { exists } from "@local/utils";
import { hasObjectChanged } from "../../general";
import { createRel } from "./createRel";
const findConnectedItems = (original, updated, idField = "id") => {
    const connected = [];
    for (const updatedItem of updated) {
        if (!updatedItem || !updatedItem[idField])
            continue;
        const originalItem = original.find(item => item[idField] === updatedItem[idField]);
        if (!originalItem)
            connected.push(updatedItem[idField]);
    }
    return connected.length > 0 ? connected : undefined;
};
const findCreatedItems = (original, updated, relation, shape, preShape) => {
    const idField = shape.idField ?? "id";
    const preShaper = preShape ?? ((x) => x);
    const originalDataArray = asArray(original[relation]);
    const updatedDataArray = asArray(updated[relation]);
    const createdItems = [];
    for (const updatedItem of updatedDataArray) {
        if (!updatedItem || !updatedItem[idField])
            continue;
        const oi = originalDataArray.find(item => item[idField] === updatedItem[idField]);
        if (!oi && Object.values(updatedItem).length > 1) {
            createdItems.push(shape.create(preShaper(updatedItem, updated)));
        }
    }
    return createdItems.length > 0 ? createdItems : undefined;
};
const findUpdatedItems = (original, updated, relation, shape, preShape) => {
    const idField = shape.idField ?? "id";
    const preShaper = preShape ?? ((x) => x);
    const originalDataArray = asArray(original[relation]);
    const updatedDataArray = asArray(updated[relation]);
    const updatedItems = [];
    for (const updatedItem of updatedDataArray) {
        if (!updatedItem || !updatedItem[idField])
            continue;
        const oi = originalDataArray.find(item => item && item[idField] && item[idField] === updatedItem[idField]);
        if (oi && (shape.hasObjectChanged ?? hasObjectChanged)(preShaper(oi, original), preShaper(updatedItem, updated))) {
            updatedItems.push(shape.update(preShaper(oi, original), preShaper(updatedItem, updated)));
        }
    }
    return updatedItems.length > 0 ? updatedItems : undefined;
};
const findDeletedItems = (original, updated, idField = "id") => {
    const removed = [];
    for (const originalItem of original) {
        if (!originalItem || !originalItem[idField])
            continue;
        const updatedItem = updated.find(item => item[idField] === originalItem[idField]);
        if (!updatedItem)
            removed.push(originalItem[idField]);
    }
    return removed.length > 0 ? removed : undefined;
};
const findDisconnectedItems = (original, updated, idField = "id") => {
    const disconnected = [];
    for (const originalItem of original) {
        if (!original || !originalItem[idField])
            continue;
        const updatedItem = updated.find(item => item[idField] === originalItem[idField]);
        if (!updatedItem)
            disconnected.push(originalItem[idField]);
    }
    return disconnected.length > 0 ? disconnected : undefined;
};
const asArray = (value) => {
    if (Array.isArray(value))
        return value;
    return [value];
};
export const updateRel = (original, updated, relation, relTypes, isOneToOne, shape, preShape) => {
    if (relTypes.includes("Create") || relTypes.includes("Update")) {
        if (!shape)
            throw new Error(`Model is required if relTypes includes "Create" or "Update": ${relation}`);
    }
    const originalRelationData = original[relation];
    const updatedRelationData = updated[relation];
    if (!exists(originalRelationData)) {
        return createRel(updated, relation, relTypes.filter(x => x === "Create" || x === "Connect"), isOneToOne, shape);
    }
    if (updatedRelationData === undefined)
        return {};
    const filteredRelTypes = updatedRelationData === null ?
        relTypes.filter(x => x !== "Create" && x !== "Connect" && x !== "Update") :
        relTypes;
    const result = {};
    const idField = shape?.idField ?? "id";
    for (const t of filteredRelTypes) {
        if (t === "Connect") {
            const shaped = findConnectedItems(asArray(originalRelationData), asArray(updatedRelationData), idField);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
        else if (t === "Create") {
            const shaped = findCreatedItems(original, updated, relation, shape, preShape);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
        else if (t === "Delete") {
            const shaped = findDeletedItems(asArray(originalRelationData), asArray(updatedRelationData), idField);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
        else if (t === "Disconnect") {
            const shaped = findDisconnectedItems(asArray(originalRelationData), asArray(updatedRelationData), idField);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
        else if (t === "Update") {
            const shaped = findUpdatedItems(original, updated, relation, shape, preShape);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
    }
    return result;
};
//# sourceMappingURL=updateRel.js.map