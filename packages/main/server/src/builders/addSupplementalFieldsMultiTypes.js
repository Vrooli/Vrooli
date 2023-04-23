import { addSupplementalFields } from "./addSupplementalFields";
export const addSupplementalFieldsMultiTypes = async (data, partial, prisma, userData) => {
    const combinedData = [];
    const combinedPartialInfo = [];
    for (const key in data) {
        for (const item of data[key]) {
            combinedData.push(item);
            combinedPartialInfo.push(partial[key]);
        }
    }
    const combinedResult = await addSupplementalFields(prisma, userData, combinedData, combinedPartialInfo);
    const formatted = {};
    let start = 0;
    for (const key in data) {
        const end = start + data[key].length;
        formatted[key] = combinedResult.slice(start, end);
        start = end;
    }
    return formatted;
};
//# sourceMappingURL=addSupplementalFieldsMultiTypes.js.map