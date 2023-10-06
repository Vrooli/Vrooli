import { uuidValidate } from "@local/shared";
import { shapeHelper, ShapeHelperInput, ShapeHelperOutput, ShapeHelperProps } from "../../builders/shapeHelper";
import { RelationshipType } from "../../builders/types";

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
    // Make sure that all tag relations are objects instead of strings
    const keys = ["Create", "Connect", "Delete", "Disconnect", "Update"].map(op => `${relation}${op}` as string);
    const tagsToObject = (tags: (string | object)[]) => {
        return tags.map(t => typeof t === "string" ? uuidValidate(t) ? { id: t } : { tag: t } : t);
    };
    keys.forEach(key => {
        if (Array.isArray(data[key])) {
            data[key] = tagsToObject(data[key]);
        }
    });
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
