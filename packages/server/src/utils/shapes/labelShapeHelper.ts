import { shapeHelper, ShapeHelperOutput, ShapeHelperProps } from "../../builders/shapeHelper.js";
import { RelationshipType } from "../../builders/types.js";
import { DbProvider } from "../../db/provider.js";

// Types of objects which have labels
type LabelledObjectType = "Api" | "Chat" | "Code" | "FocusMode" | "Issue" | "Meeting" | "Note" | "Project" | "Routine" | "Schedule" | "Standard";

/**
 * Maps type of a label's parent with the unique field
 */
const parentMapper: { [key in LabelledObjectType]: string } = {
    "Api": "api_labels_labelledid_labelid_unique",
    "Chat": "chat_labels_labelledid_labelid_unique",
    "Code": "code_labels_labelledid_labelid_unique",
    "FocusMode": "focus_mode_labels_labelledid_labelid_unique",
    "Issue": "issue_labels_labelledid_labelid_unique",
    "Meeting": "meeting_labels_labelledid_labelid_unique",
    "Note": "note_labels_labelledid_labelid_unique",
    "Project": "project_labels_labelledid_labelid_unique",
    "Routine": "routine_labels_labelledid_labelid_unique",
    "Schedule": "schedule_labels_labelledid_labelid_unique",
    "Standard": "standard_labels_labelledid_labelid_unique",
};

type LabelShapeHelperProps<
    Types extends readonly RelationshipType[],
> = {
    parentType: LabelledObjectType;
    relation?: string;
} & Omit<ShapeHelperProps<false, Types, false>, "isOneToOne" | "joinData" | "objectType" | "parentRelationshipName" | "relation" | "softDelete">;

/**
* Add, update, or remove label data for an object.
*/
export async function labelShapeHelper<
    Types extends readonly RelationshipType[],
>({
    data,
    parentType,
    relation = "labels",
    ...rest
}: LabelShapeHelperProps<Types>):
    Promise<ShapeHelperOutput<false, "id">> {
    // Labels get special logic because they are treated as strings in GraphQL, 
    // instead of a normal relationship object
    // If any label creates/connects, make sure they exist/not exist
    const initialCreate = Array.isArray(data[`${relation}Create` as string]) ?
        data[`${relation}Create` as string].map((c: { id: string }) => c.id) :
        typeof data[`${relation}Create` as string] === "object" ? [data[`${relation}Create` as string].id] :
            [];
    const initialConnect = Array.isArray(data[`${relation}Connect` as string]) ?
        data[`${relation}Connect` as string] :
        typeof data[`${relation}Connect` as string] === "object" ? [data[`${relation}Connect` as string]] :
            [];
    const initialCombined = [...initialCreate, ...initialConnect];
    if (initialCombined.length > 0) {
        // Query for all of the labels, to determine which ones exist
        const existing = await DbProvider.get().label.findMany({
            where: { label: { in: initialCombined } },
            select: { id: true },
        });
        // All existing labels are the new connects
        data[`${relation}Connect` as string] = existing.map((t) => ({ id: t.id }));
        // All new labels are the new creates
        data[`${relation}Create` as string] = initialCombined.filter((t) => !existing.some((et) => et.id === t)).map((t) => typeof t === "string" ? ({ id: t }) : t);
    }
    // Shape disconnects and deletes
    if (Array.isArray(data[`${relation}Disconnect` as string])) {
        data[`${relation}Disconnect` as string] = data[`${relation}Disconnect` as string].map((t: string | { id: string }) => typeof t === "string" ? ({ id: t }) : t);
    }
    if (Array.isArray(data[`${relation}Delete` as string])) {
        data[`${relation}Delete` as string] = data[`${relation}Delete` as string].map((t: string | { id: string }) => typeof t === "string" ? ({ id: t }) : t);
    }
    return shapeHelper({
        data,
        isOneToOne: false,
        joinData: {
            fieldName: "label",
            uniqueFieldName: parentMapper[parentType],
            childIdFieldName: "labelId",
            parentIdFieldName: "labelledId",
            parentId: data.id ?? null,
        },
        objectType: "Label",
        parentRelationshipName: "",
        relation,
        ...rest,
    });
}
