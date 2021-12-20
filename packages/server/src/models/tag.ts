import { Organization, Project, Routine, Standard, Tag, TagInput, User } from "schema/types";
import { RecursivePartial } from "types";
import { addJoinTables, BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, removeJoinTables, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type TagRelationshipList = 'organizations' | 'projects' | 'routines' | 'standards' |
    'starredBy' | 'votes';
// Type 2. QueryablePrimitives
export type TagQueryablePrimitives = Omit<Tag, TagRelationshipList>;
// Type 3. AllPrimitives
export type TagAllPrimitives = TagQueryablePrimitives;
// type 4. FullModel
export type TagFullModel = TagAllPrimitives &
{
    organizations: { tagged: Organization[] },
    projects: { tagged: Project[] },
    routines: { tagged: Routine[] },
    standards: { tagged: Standard[] },
    starredBy: { user: User[] },
    votes: { voter: User[] },
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<Tag, TagFullModel> => {
    const joinMapper = {
        organizations: 'tagged',
        projects: 'tagged',
        routines: 'tagged',
        standards: 'tagged',
        starredBy: 'user',
        votes: 'voter',
    };
    return {
        toDB: (obj: RecursivePartial<Tag>): RecursivePartial<TagFullModel> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<TagFullModel>): RecursivePartial<Tag> => removeJoinTables(obj, joinMapper)
    }
}

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function TagModel(prisma: any) {
    let obj: BaseState<Tag> = {
        prisma,
        model: MODEL_TYPES.Tag,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Tag>(obj),
        ...creater<TagInput, Tag>(obj),
        ...formatter(),
        ...updater<TagInput, Tag>(obj),
        ...deleter(obj),
        ...reporter()
    }
}

//==============================================================
/* #endregion Model */
//==============================================================