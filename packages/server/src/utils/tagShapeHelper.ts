import { shapeHelper, ShapeHelperInput, ShapeHelperOutput, ShapeHelperProps } from "../builders";
import { RelationshipType } from "../builders/types";

// Types of objects which have tags
type TaggedObjectType = "Api" | "Note" | "Organization" | "Post" | "Project" | "Question" | "Routine" | "SmartContract" | "Standard";

/**
 * Maps type of a tag's parent with the unique field
 */
const parentMapper: { [key in TaggedObjectType]: string } = {
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

type TagShapeHelperProps<
    Input extends ShapeHelperInput<false, false, Types[number], FieldName>,
    Types extends readonly RelationshipType[],
    FieldName extends string,
> = {
    parentType: TaggedObjectType;
    relation: FieldName;
} & Omit<ShapeHelperProps<Input, false, false, Types, FieldName, "tag", false>, "isRequired" | "isOneToOne" | "joinData" | "objectType" | "parentRelationshipName" | "primaryKey" | "relation" | "softDelete">;

/**
* Add, update, or remove tag data for an object.
*/
export const tagShapeHelper = async <
    Types extends readonly RelationshipType[],
    Input extends ShapeHelperInput<false, false, Types[number], FieldName>,
    FieldName extends string,
>({
    data,
    parentType,
    prisma,
    relation,
    ...rest
}: TagShapeHelperProps<Input, Types, FieldName>):
    Promise<ShapeHelperOutput<false, false, Types[number], any, "tag">> => { // Can't specify FieldName in output because ShapeHelperOutput doesn't support join tables. The expected fieldName is the unique field name, which is found inside this function
    // Define keys
    const createKey = `${relation}Create` as string;
    const connectKey = `${relation}Connect` as string;
    const deleteKey = `${relation}Delete` as string;
    const disconnectKey = `${relation}Disconnect` as string;
    // Helper function to ensure that tags are an object
    const tagsToObject = (tags: any[]) =>
        tags.map((t: any) => typeof t === "string" ? ({ tag: t }) : t);
    // Tags get special logic because they are treated as strings in GraphQL, 
    // instead of a normal relationship object
    // If any tag creates/connects, make sure they exist/not exist
    const initialCreate = Array.isArray(data[createKey]) ?
        data[createKey].map((c: any) => c.tag) :
        typeof data[createKey] === "object" ? [data[createKey].tag] :
            [];
    const initialConnect = Array.isArray(data[connectKey]) ?
        data[connectKey] :
        typeof data[connectKey] === "object" ? [data[connectKey]] :
            [];
    const initialCombined = [...initialCreate, ...initialConnect];
    if (initialCombined.length > 0) {
        // Query for all of the tags, to determine which ones exist
        const existing = await prisma.tag.findMany({
            where: { tag: { in: initialCombined } },
            select: { tag: true },
        });
        // All existing tags are the new connects
        data[connectKey] = existing.map((t) => ({ tag: t.tag }));
        // All new tags are the new creates
        data[createKey] = initialCombined.filter((t) => !existing.some((et) => et.tag === t)).map((t) => typeof t === "string" ? ({ tag: t }) : t);
    }
    // Shape disconnects and deletes
    if (Array.isArray(data[disconnectKey])) {
        data[disconnectKey] = tagsToObject(data[disconnectKey]);
    }
    if (Array.isArray(data[deleteKey])) {
        data[deleteKey] = tagsToObject(data[deleteKey]);
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
            parentId: (data as any).id ?? null,
        },
        objectType: "Tag",
        parentRelationshipName: "",
        primaryKey: "tag",
        prisma,
        relation,
        ...rest,
    });
};
