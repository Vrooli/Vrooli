import { uuidValidate } from "@local/shared";
import { shapeHelper, ShapeHelperOutput, ShapeHelperProps } from "../../builders/shapeHelper";
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
    Types extends readonly RelationshipType[],
> = {
    parentType: TaggedObjectType;
    relation?: string;
} & Omit<ShapeHelperProps<false, Types, false>, "isOneToOne" | "joinData" | "objectType" | "parentRelationshipName" | "relation" | "softDelete">;

/**
* Add, update, or remove tag data for an object.
*/
export const tagShapeHelper = async <
    Types extends readonly RelationshipType[],
>({
    data,
    parentType,
    relation = "tags",
    ...rest
}: TagShapeHelperProps<Types>):
    Promise<ShapeHelperOutput<false, "tag">> => {
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
        joinData: {
            fieldName: "tag",
            uniqueFieldName: parentMapper[parentType],
            childIdFieldName: "tagTag",
            parentIdFieldName: "taggedId",
            parentId: data.id ?? null,
        },
        objectType: "Tag",
        parentRelationshipName: "",
        relation,
        ...rest,
    });
};
