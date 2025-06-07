import { validatePK } from "@vrooli/shared";
import { shapeHelper, type ShapeHelperOutput, type ShapeHelperProps } from "../../builders/shapeHelper.js";
import { type RelationshipType } from "../../builders/types.js";

// Types of objects which have tags
type TaggedObjectType = "Resource" | "Team";

/**
 * Maps type of a tag's parent with the unique field
 */
const parentMapper: { [key in TaggedObjectType]: string } = {
    "Resource": "resource_tags_taggedid_tagTag_unique",
    "Team": "team_tags_taggedid_tagTag_unique",
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
export async function tagShapeHelper<
    Types extends readonly RelationshipType[],
>({
    data,
    parentType,
    relation = "tags",
    ...rest
}: TagShapeHelperProps<Types>):
    Promise<ShapeHelperOutput<false, "tag">> {
    // Make sure that all tag relations are objects instead of strings
    const keys = ["Create", "Connect", "Delete", "Disconnect", "Update"].map(op => `${relation}${op}` as string);
    function tagsToObject(tags: (string | object)[]) {
        return tags.map(t => typeof t === "string" ? validatePK(t) ? { id: t } : { tag: t } : t);
    }
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
}
