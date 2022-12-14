import { shapeHelper, ShapeHelperInput, ShapeHelperOutput, ShapeHelperProps } from "../builders";
import { RelationshipType } from "../builders/types";

// Types of objects which have tags
type TaggedObjectType = 'Api' | 'Note' | 'Organization' | 'Post' | 'Project' | 'Routine' | 'SmartContract' | 'Standard';

/**
 * Maps type of a tag's parent with the unique field
 */
const parentMapper: { [key in TaggedObjectType]: string } = {
    'Api': 'api_tags_taggedid_tagTag_unique',
    'Note': 'note_tags_taggedid_tagTag_unique',
    'Organization': 'organization_tags_taggedid_tagTag_unique',
    'Post': 'post_tags_taggedid_tagTag_unique',
    'Project': 'project_tags_taggedid_tagTag_unique',
    'Routine': 'routine_tags_taggedid_tagTag_unique',
    'SmartContract': 'smart_contract_tags_taggedid_tagTag_unique',
    'Standard': 'standard_tags_taggedid_tagTag_unique',
}

type TagShapeHelperProps<
    Input extends ShapeHelperInput<false, IsRequired, Types[number], RelField>,
    IsRequired extends boolean,
    Types extends readonly RelationshipType[],
    RelField extends string,
> = {
    parentType: TaggedObjectType;
    relation: string;
} & Omit<ShapeHelperProps<Input, false, IsRequired, Types, RelField, 'tag', false>, 'joinData' | 'objectType' | 'parentRelationshipName' | 'primaryKey' | 'softDelete'>;

/**
* Add, update, or remove tag data for an object.
*/
export const tagShapeHelper = async <
    IsRequired extends boolean,
    Types extends readonly RelationshipType[],
    RelField extends string,
    Input extends ShapeHelperInput<false, IsRequired, Types[number], RelField>,
>({
    data,
    isRequired,
    parentType,
    prisma,
    relation,
    relTypes,
    userData,
}: TagShapeHelperProps<Input, IsRequired, Types, RelField>):
    Promise<ShapeHelperOutput<false, IsRequired, Types[number], RelField, 'tag'>> => {
    // Tags get special logic because they are treated as strings in GraphQL, 
    // instead of a normal relationship object
    // If any tag creates/connects, make sure they exist/not exist
    const initialCreate = Array.isArray(data[`${relation}Create` as string]) ?
        data[`${relation}Create` as string].map((c: any) => c.tag) :
        typeof data[`${relation}Create` as string] === 'object' ? [data[`${relation}Create` as string].tag] :
            [];
    const initialConnect = Array.isArray(data[`${relation}Connect` as string]) ?
        data[`${relation}Connect` as string] :
        typeof data[`${relation}Connect` as string] === 'object' ? [data[`${relation}Connect` as string]] :
            [];
    const initialCombined = [...initialCreate, ...initialConnect];
    if (initialCombined.length > 0) {
        // Query for all of the tags, to determine which ones exist
        const existing = await prisma.tag.findMany({
            where: { tag: { in: initialCombined } },
            select: { tag: true }
        });
        // All existing tags are the new connects
        data[`${relation}Connect` as string] = existing.map((t) => ({ tag: t.tag }));
        // All new tags are the new creates
        data[`${relation}Create` as string] = initialCombined.filter((t) => !existing.some((et) => et.tag === t)).map((t) => typeof t === 'string' ? ({ tag: t }) : t);
    }
    // Shape disconnects and deletes
    if (Array.isArray(data[`${relation}Disconnect` as string])) {
        data[`${relation}Disconnect` as string] = data[`${relation}Disconnect` as string].map((t: any) => typeof t === 'string' ? ({ tag: t }) : t);
    }
    if (Array.isArray(data[`${relation}Delete` as string])) {
        data[`${relation}Delete` as string] = data[`${relation}Delete` as string].map((t: any) => typeof t === 'string' ? ({ tag: t }) : t);
    }
    return shapeHelper({
        data,
        isOneToOne: false,
        isRequired,
        joinData: {
            fieldName: 'tag',
            uniqueFieldName: parentMapper[parentType],
            childIdFieldName: 'tagTag',
            parentIdFieldName: 'taggedId',
            parentId: (data as any).id ?? null,
        },
        objectType: 'Tag',
        parentRelationshipName: '',
        primaryKey: 'tag',
        prisma,
        relation,
        relTypes,
        userData,
    })
}