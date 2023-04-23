import { shapeHelper } from "../builders";
const parentMapper = {
    "Api": "api_tags_taggedid_tagTag_unique",
    "Note": "note_tags_taggedid_tagTag_unique",
    "Organization": "organization_tags_taggedid_tagTag_unique",
    "Post": "post_tags_taggedid_tagTag_unique",
    "Project": "project_tags_taggedid_tagTag_unique",
    "Question": "question_tags_taggedid_tagTag_unique",
    "Routine": "routine_tags_taggedid_tagTag_unique",
    "SmartContract": "smart_contract_tags_taggedid_tagTag_unique",
    "Standard": "standard_tags_taggedid_tagTag_unique",
};
export const tagShapeHelper = async ({ data, parentType, prisma, relation, ...rest }) => {
    const initialCreate = Array.isArray(data[`${relation}Create`]) ?
        data[`${relation}Create`].map((c) => c.tag) :
        typeof data[`${relation}Create`] === "object" ? [data[`${relation}Create`].tag] :
            [];
    const initialConnect = Array.isArray(data[`${relation}Connect`]) ?
        data[`${relation}Connect`] :
        typeof data[`${relation}Connect`] === "object" ? [data[`${relation}Connect`]] :
            [];
    const initialCombined = [...initialCreate, ...initialConnect];
    if (initialCombined.length > 0) {
        const existing = await prisma.tag.findMany({
            where: { tag: { in: initialCombined } },
            select: { tag: true },
        });
        data[`${relation}Connect`] = existing.map((t) => ({ tag: t.tag }));
        data[`${relation}Create`] = initialCombined.filter((t) => !existing.some((et) => et.tag === t)).map((t) => typeof t === "string" ? ({ tag: t }) : t);
    }
    if (Array.isArray(data[`${relation}Disconnect`])) {
        data[`${relation}Disconnect`] = data[`${relation}Disconnect`].map((t) => typeof t === "string" ? ({ tag: t }) : t);
    }
    if (Array.isArray(data[`${relation}Delete`])) {
        data[`${relation}Delete`] = data[`${relation}Delete`].map((t) => typeof t === "string" ? ({ tag: t }) : t);
    }
    return shapeHelper({
        data,
        isOneToOne: false,
        isRequired: false,
        joinData: {
            fieldName: "tag",
            uniqueFieldName: parentMapper[parentType],
            childIdFieldName: "tagTag",
            parentIdFieldName: "taggedId",
            parentId: data.id ?? null,
        },
        objectType: "Tag",
        parentRelationshipName: "",
        primaryKey: "tag",
        prisma,
        relation,
        ...rest,
    });
};
//# sourceMappingURL=tagShapeHelper.js.map