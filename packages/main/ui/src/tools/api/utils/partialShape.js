import { exists } from "@local/utils";
import { uniqueFragmentName } from "./uniqueFragmentName";
import { unlazy, unlazyDeep } from "./unlazy";
const addFragments = (fragmentsByShape, define) => {
    let result = { ...fragmentsByShape } ?? {};
    for (const partial of Object.values(define ?? {})) {
        if (!exists(partial.__selectionType) || !exists(partial.__typename)) {
            console.error(`Error: __selectionType or __typename is not defined for a fragment ${partial.__typename}`, partial);
            continue;
        }
        const actualKey = uniqueFragmentName(partial.__typename, partial.__selectionType);
        if (!exists(result[actualKey])) {
            const { __define, ...rest } = partial;
            result[actualKey] = rest;
            result = { ...result, ...__define };
        }
    }
    return result;
};
export const partialShape = async (selection, lastDefine = {}) => {
    const result = {};
    const data = await unlazyDeep(selection);
    let uniqueFragments = {};
    uniqueFragments = addFragments(uniqueFragments, data.__define);
    let currDefine = { ...(data.__define ?? {}) };
    if (Object.keys(currDefine).length === 0)
        currDefine = { ...lastDefine };
    for (const key of Object.keys(data)) {
        if (["__typename", "__define", "__selectionType"].includes(key))
            continue;
        if (key === "__union") {
            if (!exists(result.__union)) {
                result.__union = {};
            }
            for (const [unionKey, value] of Object.entries(data.__union)) {
                if (typeof value === "string" || typeof value === "number") {
                    if (!exists(currDefine[value]))
                        continue;
                    const defineData = currDefine[value];
                    result.__union[unionKey] = uniqueFragmentName(defineData.__typename, defineData.__selectionType);
                }
                else {
                    const { __define, ...rest } = await unlazy(value);
                    uniqueFragments = addFragments(uniqueFragments, __define);
                    result.__union[unionKey] = { ...rest };
                }
            }
        }
        else if (exists(data[key]?.__union)) {
            if (!exists(result[key])) {
                result[key] = { __union: {} };
            }
            for (const [unionKey, value] of Object.entries(data[key].__union)) {
                if (typeof value === "string" || typeof value === "number") {
                    if (!exists(currDefine[value]))
                        continue;
                    const defineData = currDefine[value];
                    result[key].__union[unionKey] = uniqueFragmentName(defineData.__typename, defineData.__selectionType);
                }
                else {
                    const { __define, ...rest } = await unlazy(value);
                    uniqueFragments = addFragments(uniqueFragments, __define);
                    result[key].__union[unionKey] = { ...rest };
                }
            }
        }
        else if (exists(data[key]?.__use)) {
            const useKey = data[key].__use;
            if (exists(currDefine[useKey])) {
                const defineData = currDefine[useKey];
                result[key] = { __typename: key, __use: uniqueFragmentName(defineData.__typename, defineData.__selectionType) };
            }
            if (Object.keys(data[key]).filter(k => k !== "__use" && k !== "__typename").length > 0) {
                const { __define, __use, ...rest } = await unlazy(data[key]);
                uniqueFragments = addFragments(uniqueFragments, __define);
                result[key] = { ...result[key], ...rest };
            }
        }
        else {
            if (!exists(data[key]))
                continue;
            if (typeof data[key] === "boolean") {
                result[key] = true;
            }
            else {
                const { __define, ...rest } = await partialShape(data[key] ?? {}, currDefine);
                uniqueFragments = addFragments(uniqueFragments, __define);
                result[key] = rest;
            }
        }
    }
    if (Object.keys(uniqueFragments).length > 0)
        result.__define = uniqueFragments;
    return result;
};
//# sourceMappingURL=partialShape.js.map