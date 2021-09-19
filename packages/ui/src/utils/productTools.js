// Functions for easy manipulation of product data

import { addToArray, updateArray } from "./arrayTools";
import { updateObject } from "./objectTools";

export const getProductTrait = (traitName, productData) => {
    if (!(typeof traitName === 'string')) return null;
    const lowered = traitName.toLowerCase();
    return productData?.traits ? productData.traits.find(t => t.name.toLowerCase() === lowered)?.value : null;
}

export const setProductTrait = (name, value, productData, createIfNotExists=false) => {
    if (!productData?.traits) return null;
    if (!(typeof name === 'string')) return null;
    const lowered = name.toLowerCase();
    const traitIndex = productData.traits.findIndex(t => t?.name?.toLowerCase() === lowered);
    if (traitIndex < 0 && !createIfNotExists) return null;
    const updatedTraits = traitIndex < 0 ?
        addToArray(productData.traits, { name, value }):
        updateArray(productData.traits, traitIndex, { name, value });
    return updateObject(productData, 'traits', updatedTraits);
}

export const setProductSkuField = (fieldName, index, value, productData) => {
    if (!Array.isArray(productData?.skus)) return null;
    if (index < 0 || index >= productData.skus.length) return null;
    const updatedSku = updateObject(productData.skus[index], fieldName, value);
    const updatedSkus = updateArray(productData.skus, index, updatedSku);
    return updateObject(productData, 'skus', updatedSkus);
}