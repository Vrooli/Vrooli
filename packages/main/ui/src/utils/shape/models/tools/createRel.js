export const createRel = (item, relation, relTypes, isOneToOne, shape, preShape) => {
    if (relTypes.includes("Create")) {
        if (!shape)
            throw new Error(`Model is required if relTypes includes "Create": ${relation}`);
    }
    const relationData = item[relation];
    if (relationData === undefined)
        return {};
    if (relationData === null)
        return {};
    const result = {};
    const preShaper = preShape ?? ((x) => x);
    for (const t of relTypes) {
        if (t === "Connect") {
            let filteredRelationData = Array.isArray(relationData) ? relationData : [relationData];
            filteredRelationData = relTypes.includes("Create") ?
                filteredRelationData.filter((x) => Object.keys(x).every((k) => ["id", "__typename"].includes(k))) :
                filteredRelationData;
            if (filteredRelationData.length === 0)
                continue;
            result[`${relation}${t}`] = isOneToOne === "one" ?
                relationData[shape?.idField ?? "id"] :
                relationData.map((x) => x[shape?.idField ?? "id"]);
        }
        else if (t === "Create") {
            let filteredRelationData = Array.isArray(relationData) ? relationData : [relationData];
            filteredRelationData = filteredRelationData.filter((x) => Object.keys(x).some((k) => !["id", "__typename"].includes(k)));
            if (filteredRelationData.length === 0)
                continue;
            result[`${relation}${t}`] = isOneToOne === "one" ?
                shape.create(preShaper(relationData)) :
                relationData.map((x) => shape.create(preShaper(x)));
        }
    }
    return result;
};
//# sourceMappingURL=createRel.js.map