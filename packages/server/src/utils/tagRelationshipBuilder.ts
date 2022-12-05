import { relationshipBuilderHelper } from "../builders";
import { BuiltRelationship } from "../builders/types";
import { SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";

// Types of objects which have tags
type TaggedObjectType = 'Api' | 'Organization' | 'Post' | 'Project' | 'Routine' | 'SmartContract' | 'Standard';

/**
 * Maps type of a tag's parent with the unique field
 */
const parentMapper: { [key in TaggedObjectType]: string } = {
    'Api': 'api_tags_taggedid_tagTag_unique',
    'Organization': 'organization_tags_taggedid_tagTag_unique',
    'Post': 'post_tags_taggedid_tagTag_unique',
    'Project': 'project_tags_taggedid_tagTag_unique',
    'Routine': 'routine_tags_taggedid_tagTag_unique',
    'SmartContract': 'smart_contract_tags_taggedid_tagTag_unique',
    'Standard': 'standard_tags_taggedid_tagTag_unique',
}

/**
* Add, update, or remove tag data for an object.
*/
export const tagRelationshipBuilder = async <IsAdd extends boolean>(
    prisma: PrismaType,
    userData: SessionUser,
    data: { [x: string]: any },
    parentType: TaggedObjectType,
    isAdd = true as IsAdd,
    relationshipName: string = 'tags',
): Promise<BuiltRelationship<any, IsAdd, false, any, any>> => {
    // Tags get special logic because they are treated as strings in GraphQL, 
    // instead of a normal relationship object
    // If any tag creates/connects, make sure they exist/not exist
    const initialCreate = Array.isArray(data[`${relationshipName}Create`]) ?
        data[`${relationshipName}Create`].map((c: any) => c.tag) :
        typeof data[`${relationshipName}Create`] === 'object' ? [data[`${relationshipName}Create`].tag] :
            [];
    const initialConnect = Array.isArray(data[`${relationshipName}Connect`]) ?
        data[`${relationshipName}Connect`] :
        typeof data[`${relationshipName}Connect`] === 'object' ? [data[`${relationshipName}Connect`]] :
            [];
    const initialCombined = [...initialCreate, ...initialConnect];
    if (initialCombined.length > 0) {
        // Query for all of the tags, to determine which ones exist
        const existing = await prisma.tag.findMany({
            where: { tag: { in: initialCombined } },
            select: { tag: true }
        });
        // All existing tags are the new connects
        data[`${relationshipName}Connect`] = existing.map((t) => ({ tag: t.tag }));
        // All new tags are the new creates
        data[`${relationshipName}Create`] = initialCombined.filter((t) => !existing.some((et) => et.tag === t)).map((t) => typeof t === 'string' ? ({ tag: t }) : t);
    }
    // Shape disconnects and deletes
    if (Array.isArray(data[`${relationshipName}Disconnect`])) {
        data[`${relationshipName}Disconnect`] = data[`${relationshipName}Disconnect`].map((t: any) => typeof t === 'string' ? ({ tag: t }) : t);
    }
    if (Array.isArray(data[`${relationshipName}Delete`])) {
        data[`${relationshipName}Delete`] = data[`${relationshipName}Delete`].map((t: any) => typeof t === 'string' ? ({ tag: t }) : t);
    }
    return relationshipBuilderHelper<any, IsAdd, false, any, any, any, any>({
        data,
        isAdd,
        idField: 'tag',
        isOneToOne: false,
        isTransferable: false,
        joinData: {
            fieldName: 'tag',
            uniqueFieldName: parentMapper[parentType],
            childIdFieldName: 'tagTag',
            parentIdFieldName: 'taggedId',
            parentId: data.id ?? null,
        },
        prisma,
        relationshipName,
        userData,
    });
}