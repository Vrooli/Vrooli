import { shapeHelper } from "../builders";
const parentMapper = {
    "Api": "api_labels_labelledid_labelid_unique",
    "FocusMode": "focus_mode_labels_labelledid_labelid_unique",
    "Issue": "issue_labels_labelledid_labelid_unique",
    "Meeting": "meeting_labels_labelledid_labelid_unique",
    "Note": "note_labels_labelledid_labelid_unique",
    "Project": "project_labels_labelledid_labelid_unique",
    "Routine": "routine_labels_labelledid_labelid_unique",
    "SmartContract": "smart_contract_labels_labelledid_labelid_unique",
    "Standard": "standard_labels_labelledid_labelid_unique",
};
export const labelShapeHelper = async ({ data, parentType, prisma, relation, ...rest }) => {
    const initialCreate = Array.isArray(data[`${relation}Create`]) ?
        data[`${relation}Create`].map((c) => c.id) :
        typeof data[`${relation}Create`] === "object" ? [data[`${relation}Create`].id] :
            [];
    const initialConnect = Array.isArray(data[`${relation}Connect`]) ?
        data[`${relation}Connect`] :
        typeof data[`${relation}Connect`] === "object" ? [data[`${relation}Connect`]] :
            [];
    const initialCombined = [...initialCreate, ...initialConnect];
    if (initialCombined.length > 0) {
        const existing = await prisma.label.findMany({
            where: { label: { in: initialCombined } },
            select: { id: true },
        });
        data[`${relation}Connect`] = existing.map((t) => ({ id: t.id }));
        data[`${relation}Create`] = initialCombined.filter((t) => !existing.some((et) => et.id === t)).map((t) => typeof t === "string" ? ({ id: t }) : t);
    }
    if (Array.isArray(data[`${relation}Disconnect`])) {
        data[`${relation}Disconnect`] = data[`${relation}Disconnect`].map((t) => typeof t === "string" ? ({ id: t }) : t);
    }
    if (Array.isArray(data[`${relation}Delete`])) {
        data[`${relation}Delete`] = data[`${relation}Delete`].map((t) => typeof t === "string" ? ({ id: t }) : t);
    }
    return shapeHelper({
        data,
        isOneToOne: false,
        isRequired: false,
        joinData: {
            fieldName: "label",
            uniqueFieldName: parentMapper[parentType],
            childIdFieldName: "labelId",
            parentIdFieldName: "labelledId",
            parentId: data.id ?? null,
        },
        objectType: "Label",
        parentRelationshipName: "",
        primaryKey: "id",
        prisma,
        relation,
        ...rest,
    });
};
//# sourceMappingURL=labelShapeHelper.js.map