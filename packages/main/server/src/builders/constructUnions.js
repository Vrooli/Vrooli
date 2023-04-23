import { exists, isObject } from "@local/utils";
export const constructUnions = (data, partialInfo, gqlRelMap) => {
    const resultData = data;
    const resultPartialInfo = { ...partialInfo };
    const unionFields = Object.entries(gqlRelMap).filter(([, value]) => typeof value === "object");
    for (const [gqlField, unionData] of unionFields) {
        for (const [dbField, type] of Object.entries(unionData)) {
            const isInPartialInfo = exists(resultData[dbField]);
            if (isInPartialInfo) {
                resultData[gqlField] = { ...resultData[dbField], __typename: type };
                if (isObject(resultPartialInfo[gqlField]) && isObject(resultPartialInfo[gqlField][type])) {
                    resultPartialInfo[gqlField] = resultPartialInfo[gqlField][type];
                }
            }
            delete resultData[dbField];
        }
        if (!exists(resultData[gqlField])) {
            resultData[gqlField] = null;
        }
    }
    return { data: resultData, partialInfo: resultPartialInfo };
};
//# sourceMappingURL=constructUnions.js.map