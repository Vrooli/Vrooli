import { Count, DotNotation, GqlModelType, ObjectLimit, SessionUser } from "@shared/consts";
import { PrismaType, PromiseOrValue, RecursivePartial } from "../types";
import { ObjectSchema } from 'yup';
import { PartialGraphQLInfo, PartialPrismaSelect, PrismaDelegate } from "../builders/types";
import { SortMap } from "../utils/sortMap";
import { SearchMap, SearchStringMap } from "../utils";
import { QueryAction } from "../utils/types";

export type ModelLogicType = {
    GqlCreate: Record<string, any> | undefined,
    GqlUpdate: Record<string, any> | undefined,
    GqlSearch: Record<string, any> | undefined,
    GqlSort: string | undefined,
    GqlModel: Record<string, any>,
    GqlPermission: Record<string, any>,
    PrismaCreate: Record<string, any> | undefined,
    PrismaUpdate: Record<string, any> | undefined,
    PrismaSelect: Record<string, any>,
    PrismaWhere: Record<string, any> | undefined,
    PrismaModel: Record<string, any>,
    IsTransferable?: boolean,
    IsVersioned?: boolean,
}

/**
* Basic structure of an object's business layer.
* Every business layer object has at least a PrismaType object and a format converter. 
* Everything else is optional
*/
export type ModelLogic<
    Model extends ModelLogicType,
    SuppFields extends readonly DotNotation<Model['GqlModel']>[]
> = {
    __typename: `${GqlModelType}`;
    delegate: (prisma: PrismaType) => PrismaDelegate;
    display: Displayer<Model>;
    duplicate?: Duplicator<any, any>;
    format: Formatter<Model, SuppFields>;
    search?: Model['GqlSearch'] extends undefined ? undefined :
    Model['GqlSort'] extends undefined ? undefined :
    Model['PrismaWhere'] extends undefined ? undefined :
    Searcher<{
        GqlSearch: Exclude<Model['GqlSearch'], undefined>,
        GqlSort: Exclude<Model['GqlSort'], undefined>,
        PrismaWhere: Exclude<Model['PrismaWhere'], undefined>,
    }>;
    mutate?: Mutater<{
        GqlCreate: Model['GqlCreate'],
        GqlUpdate: Model['GqlUpdate'],
        GqlModel: Exclude<Model['GqlModel'], undefined>,
        PrismaCreate: Model['PrismaCreate'],
        PrismaUpdate: Model['PrismaUpdate'],
    }>;
    validate?: Validator<{
        GqlCreate: Model['GqlCreate'],
        GqlUpdate: Model['GqlUpdate'],
        PrismaModel: Exclude<Model['PrismaModel'], undefined>,
        GqlPermission: Exclude<Model['GqlPermission'], undefined>,
        PrismaSelect: Exclude<Model['PrismaSelect'], undefined>,
        PrismaWhere: Exclude<Model['PrismaWhere'], undefined>,
        IsTransferable: Exclude<Model['IsTransferable'], undefined>,
        IsVersioned: Exclude<Model['IsVersioned'], undefined>,
    }>
} & { [x: string]: any };

/**
 * An object which can maps GraphQL fields to GqlModelTypes. Normal fields 
 * are a single GqlModelType, while unions are handled with an object. 
 * A union object maps Prisma relation fields to GqlModelTypes.
 */
export type GqlRelMap<
    GqlModel extends ModelLogicType['GqlModel'],
    PrismaModel extends ModelLogicType['PrismaModel'],
> = {
    [K in keyof GqlModel]?: `${GqlModelType}` | ({ [K2 in keyof PrismaModel]?: `${GqlModelType}` })
} & { __typename: `${GqlModelType}` };

/**
 * Allows Prisma select fields to map to GqlModelTypes
 */
export type PrismaRelMap<T> = {
    [K in keyof T]?: `${GqlModelType}`
} & { __typename: `${GqlModelType}` };

/**
 * Helper functions for adding and removing supplemental fields. These are fields 
 * are requested in the select query, but are either not in the main database or 
 * cannot be requested in the same query (e.g. isBookmarked, permissions) 
 */
export interface SupplementalConverter<
    SuppFields extends readonly any[]
> {
    /**
     * List of all supplemental fields added to the GraphQL model after the main query 
     * (i.e. all fields to be excluded)
     */
    graphqlFields: SuppFields;
    /**
     * List of all fields to add to the Prisma select query, in order to calculate 
     * supplemental fields
     */
    dbFields?: string[]; // TODO make type safer
    /**
     * Calculates supplemental fields from the main query results
     */
    toGraphQL: ({ ids, objects, partial, prisma, userData }: {
        ids: string[],
        languages: string[],
        objects: ({ id: string } & { [x: string]: any })[], // TODO: fix this type
        partial: PartialGraphQLInfo,
        prisma: PrismaType,
        userData: SessionUser | null,
    }) => Promise<{ [key in SuppFields[number]]: any[] | { [x: string]: any[] } }>;
}

type StringArrayMap<T extends readonly string[]> = {
    [K in T[number]]: true
}

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export interface Formatter<
    Model extends {
        GqlModel: ModelLogicType['GqlModel'],
        PrismaModel: ModelLogicType['PrismaModel'],
    },
    SuppFields extends readonly DotNotation<Model['GqlModel']>[]
> {
    /**
     * Maps GraphQL types to GqlModelType, with special handling for unions
     */
    gqlRelMap: GqlRelMap<Model['GqlModel'], Model['PrismaModel']>;
    /**
     * Maps Prisma types to GqlModelType
     */
    prismaRelMap: PrismaRelMap<Model['PrismaModel']>;
    /**
     * Map used to add/remove join tables from the query. 
     * Each key is a GraphQL field, and each value is the join table's relation name
     */
    joinMap?: { [key in keyof Model['GqlModel']]?: string };
    /**
     * List of fields which provide the count of a relationship. 
     * These fields must end with "Count", and exist in the GraphQL object.
     * Each field of the form {relationship}Count will be converted to
     * { [relationship]: { _count: true } } in the Prisma query
     */
    countFields: StringArrayMap<(keyof Model['GqlModel'] extends infer R ? R extends `${string}Count` ? R : never : never)[]>;
    /**
     * List of fields to always exclude from GraphQL results
     */
    hiddenFields?: string[];
    /**
     * Add join tables which are not present in GraphQL object
     */
    addJoinTables?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /**
     * Remove join tables which are not present in GraphQL object
     */
    removeJoinTables?: (data: { [x: string]: any }) => any;
    /**
     * Add _count fields
     */
    addCountFields?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /**
     * Remove _count fields
     */
    removeCountFields?: (data: { [x: string]: any }) => any;
    /**
     * Data for adding supplemental fields to the GraphQL object
     */
    supplemental?: SupplementalConverter<SuppFields>;
}

type CommonSearchFields = 'after' | 'take' | 'ids' | 'searchString';

export type SearchStringQueryParams = {
    insensitive: { contains: string; mode: 'default' | 'insensitive'; },
    languages?: string[],
    searchString: string,
}

export type SearchStringQuery<Where extends { [x: string]: any }> = {
    [x in keyof Where]: Where[x] extends infer R ?
    //Check array
    R extends { [x: string]: any }[] ?
    ((keyof typeof SearchStringMap)[] | SearchStringQuery<Where[x]>) :
    //Check object
    R extends { [x: string]: any } ?
    (keyof typeof SearchStringMap | SearchStringQuery<Where[x]>) :
    //Else
    SearchStringQuery<Where[x]> : never
}

/**
 * Describes shape of component that can be sorted in a specific order
 */
export type Searcher<
    Model extends {
        GqlSearch: Exclude<ModelLogicType['GqlSearch'], undefined>,
        GqlSort: Exclude<ModelLogicType['GqlSort'], undefined>,
        PrismaWhere: Exclude<ModelLogicType['PrismaWhere'], undefined>,
    }
> = {
    defaultSort: Model['GqlSort'];
    /**
     * Enum of all possible sort fields for this model
     * Also ensures that each field is in the SortMap object 
     * (i.e. SortMap is a superset of Model['GqlSort'])
     */
    sortBy: { [x in Model['GqlSort']]: keyof typeof SortMap };
    /**
     * Search input fields for this model
     * Also ensures that each field is in the SearchMap object
     * (i.e. SearchMap is a superset of Model['GqlSearch'])
     * 
     * NOTE: Excludes fields which are common to all models 
     * (or have special logic), such as "take", "after", etc.
     */
    searchFields: StringArrayMap<(keyof Model['GqlSearch'] extends infer R ?
        R extends keyof typeof SearchMap ?
        R extends CommonSearchFields ? never : R
        : never
        : never)[]>;
    /**
     * The maximum number of results allowed to search for at once. 
     * Defaults to 100
     */
    takeLimit?: number;
    /**
     * Query for searching by a string. To reduce code duplication, 
     * pieces of the query can be replaced with keys of the SearchStringMap object. 
     * This works for both arrays and objects
     */
    searchStringQuery: () => SearchStringQuery<Model['PrismaWhere']>;
    /**
     * Any additional data to add to the Prisma query. Not usually needed
     */
    customQueryData?: (input: Model['GqlSearch'], userData: SessionUser | null) => Model['PrismaWhere'];
}

/**
 * Describes shape of select query used to check 
 * the permissions for an object. Basically, it's the 
 * Prisma select query, but where fields can be a GqlModelType 
 * string instead of a boolean. 
 * 
 * Any field - top-level or nested - meets 
 * one of the following conditions:
 * 1. It's a partial select query matching the generic type ModelSelect
 * 2. It's a GqlModelType string, which means that object's permissions
 * will be substituted for that value
 * 3. It's a tuple of the form [GqlModelType, [omitFields]] where omitFields 
 * is a list of fields (supporting dot notation) to omit from the substitution's permissions. This 
 * is important for preventing circular references
 */
export type PermissionsMap<ModelSelect extends { [x: string]: any }> = {
    [x in keyof ModelSelect]: ModelSelect[x] | `${GqlModelType}` | [`${GqlModelType}`, string[]]
}

/**
 * Describes shape of component that has validation rules 
 */
export type Validator<
    Model extends {
        GqlCreate: ModelLogicType['GqlCreate'],
        GqlUpdate: ModelLogicType['GqlUpdate'],
        PrismaModel: Exclude<ModelLogicType['PrismaModel'], undefined>,
        GqlPermission: Exclude<ModelLogicType['GqlPermission'], undefined>,
        PrismaSelect: Exclude<ModelLogicType['PrismaSelect'], undefined>,
        PrismaWhere: Exclude<ModelLogicType['PrismaWhere'], undefined>,
        IsTransferable: Exclude<ModelLogicType['IsTransferable'], undefined>,
        IsVersioned: Exclude<ModelLogicType['IsVersioned'], undefined>,
    }
> = {
    /**
     * The maximum number of objects that can be created by a single user/organization.
     * This depends on if the owner is a user or organization, if the owner 
     * has a premium account, and if we're counting private or public objects. 
     * Accounts can also define custom limits (for a custom price), which override 
     * these defaults 
     */
    maxObjects: ObjectLimit;
    /**
     * Select query to calculate the object's permissions. This will be used - possibly in 
     * conjunction with the parent object's permissions (also queried in this field) - to determine if you 
     * are allowed to perform the mutation
     */
    permissionsSelect: (userId: string | null, languages: string[]) => PermissionsMap<Model['PrismaSelect']>;
    /**
     * Key/value pair of permission fields and resolvers to calculate them.
     */
    permissionResolvers: ({ data, isAdmin, isDeleted, isPublic }: {
        data: Model['PrismaModel'],
        isAdmin: boolean,
        isDeleted: boolean,
        isPublic: boolean,
    }) => { [x in keyof Omit<Model['GqlPermission'], 'type'>]: () => any } & { [x in Exclude<QueryAction, 'Create'> as `can${x}`]: () => boolean | Promise<boolean> };
    /**
     * Partial queries for various visibility checks
     */
    visibility: {
        /**
         * For private objects (i.e. only the owner can see them)
         */
        private: Model['PrismaWhere'];
        /**
         * For public objects (i.e. anyone can see them)
         */
        public: Model['PrismaWhere'];
        /**
         * For both private and public objects that you own
         */
        owner: (userId: string) => Model['PrismaWhere'];
    }
    /**
     * Uses query result to determine if the object is soft-deleted
     */
    isDeleted: (data: Model['PrismaModel'], languages: string[]) => boolean;
    /**
     * Uses query result to determine if the object is public. This typically means "isPrivate" and "isDeleted" are false
     */
    isPublic: (data: Model['PrismaModel'], languages: string[]) => boolean;
    /**
     * Permissions data for the object's owner
     */
    owner: (data: Model['PrismaModel']) => {
        Organization?: ({ id: string } & { [x: string]: any }) | null;
        User?: ({ id: string } & { [x: string]: any }) | null;
    }
    /**
     * String fields which must be checked for profanity. You don't need to 
     * include fields in a translate object, as those will be checked
     * automatically
     */
    profanityFields?: string[];
    /**
     * Any custom validations you want to perform before a create mutation. You must throw 
     * an error if a validation fails, since that'll return a customized error message to the
     * user
     */
    validations?: {
        common?: ({ connectMany, createMany, deleteMany, disconnectMany, languages, prisma, updateMany, userData }: {
            connectMany: string[],
            createMany: Model['GqlCreate'][],
            deleteMany: string[],
            disconnectMany: string[],
            languages: string[],
            prisma: PrismaType,
            updateMany: { where: Model['PrismaWhere'], data: Model['GqlUpdate'] }[],
            userData: SessionUser,
        }) => Promise<void> | void,
        create?: Model['GqlCreate'] extends Record<string, any> ? ({ createMany, deltaAdding, languages, prisma, userData }: {
            createMany: Model['GqlCreate'][],
            deltaAdding: number,
            languages: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void : never,
        update?: Model['GqlUpdate'] extends Record<string, any> ? ({ languages, prisma, updateMany, userData }: {
            languages: string[],
            prisma: PrismaType,
            updateMany: { where: Model['PrismaWhere'], data: Model['GqlUpdate'] }[],
            userData: SessionUser,
        }) => Promise<void> | void : never,
        connect?: ({ connectMany, languages, prisma, userData }: {
            connectMany: string[],
            languages: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void,
        delete?: ({ deleteMany, languages, prisma, userData }: {
            deleteMany: string[],
            languages: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void,
        disconnect?: ({ disconnectMany, languages, prisma, userData }: {
            disconnectMany: string[],
            languages: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void,
    }
    /**
     * Any custom transformations you want to perform before a create/update mutation, 
     * besides the ones supported by default in cudHelper. This includes converting creates to 
     * connects, which means this function has to be pretty flexible in what it allows
     */
    transformations?: {
        create?: (createMany: Model['GqlCreate'][], userId: string) => Promise<Model['GqlCreate'][]> | Model['GqlCreate'][];
        update?: (updateMany: Model['GqlUpdate'][], userId: string) => Promise<Model['GqlUpdate'][]> | Model['GqlUpdate'][];
    };
    /**
     * True if you are allowed to tranfer the object to another user/orgaization
     */
    isTransferable: Model['IsTransferable'];
} & (
        Model['IsTransferable'] extends true ? {
            /*
            * Determines if object has its original owner
            */
            hasOriginalOwner: (data: Model['PrismaModel']) => boolean;
        } : {}
    ) & (
        Model['IsVersioned'] extends true ? {
            /**
             * Determines if there is a completed version of the object
             */
            hasCompleteVersion: (data: Model['PrismaModel']) => boolean;
        } : {}
    )

/**
 * Describes shape of component that can be duplicated
 */
export type Duplicator<
    // Select must include "id" and "intendToPullRequest" fields
    Select extends { id?: boolean | undefined, intendToPullRequest?: boolean | undefined, [x: string]: any },
    Data extends { [x: string]: any }
> = {
    /**
     * Data to select from the database to duplicate. DO NOT select anything that 
     * is not meant to be duplicated 
     * 
     * 
     * NOTE: All IDs will be converted to new IDs. This is especially useful for 
     * child data that references each other, like nodes and edges 
     */
    select: Select;
    /**
     * Data to connect to new owner
     */
    owner: (id: string) => {
        Organization?: Partial<Data> | null
        User?: Partial<Data> | null;
    }
}

/**
 * Describes shape of component that can be mutated
 */
export type Mutater<Model extends {
    GqlCreate: ModelLogicType['GqlCreate'],
    GqlUpdate: ModelLogicType['GqlUpdate'],
    GqlModel: ModelLogicType['GqlModel'],
    PrismaCreate: ModelLogicType['PrismaCreate'],
    PrismaUpdate: ModelLogicType['PrismaUpdate'],
}> = {
    /**
     * Shapes data for create/update mutations, both as a main 
     * object and as a relationship object
     */
    shape: {
        /**
         * Calculates any data that requires the full context of the mutation. In other words, 
         * data that can change depending on what else is being mutated. This is useful for 
         * things like routine complexity, where the calculation depends on the complexity of subroutines.
         */
        pre?: ({ createList, updateList, deleteList, prisma, userData }: {
            createList: Model['GqlCreate'][],
            updateList: {
                where: { id: string },
                data: Model['GqlUpdate'],
            }[],
            deleteList: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<{}>,
        /**
         * Performs additional shaping after the main mutation has been performed, 
         * AND after all triggers functions have been called. This is useful 
         * for things like updating a version label, where all versions tied to that root object 
         * might need their version indexes updated. Since these versions don't appear in the 
         * create/update/delete lists, they can't be updated in the pre function.
         */
        post?: ({ created, deletedIds, updated, prisma, userData }: {
            created: (RecursivePartial<Model['GqlModel']> & { id: string })[],
            deletedIds: string[],
            updated: (RecursivePartial<Model['GqlModel']> & { id: string })[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
        create?: Model['GqlCreate'] extends Record<string, any> ?
        Model['PrismaCreate'] extends Record<string, any> ? ({ data, preMap, prisma, userData }: {
            data: Model['GqlCreate'],
            preMap: { [key in GqlModelType]?: any }, // Result of pre function on every model included in the mutation, by model type
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<Model['PrismaCreate']> : never : never,
        update?: Model['GqlUpdate'] extends Record<string, any> ?
        Model['PrismaUpdate'] extends Record<string, any> ? ({ data, preMap, prisma, userData, where }: {
            data: Model['GqlUpdate'],
            preMap: { [key in GqlModelType]?: any }, // Result of pre function on every model included in the mutation, by model type
            prisma: PrismaType,
            userData: SessionUser,
            where: { id: string },
        }) => PromiseOrValue<Model['PrismaUpdate']> : never : never
    }
    /**
     * Triggers after (or before if specified) a mutation is performed on the object. Useful for sending notifications,
     * tracking awards, and updating reputation.
     * 
     * NOTE: This is only called for top-level objects, not relationships. If you need to update 
     * data like indexes, hasCompleteVersion, etc., or initiate triggers, you should do it in the mutate.pre function
     */
    trigger?: {
        onCommon?: ({ createAuthData, created, deleted, prisma, updateAuthData, updated, updateInput, userData }: {
            createAuthData: { [id: string]: { [x: string]: any } },
            created: (RecursivePartial<Model['GqlModel']> & { id: string })[],
            deleted: Count,
            deletedIds: string[],
            updateAuthData: { [id: string]: { [x: string]: any } },
            updated: (RecursivePartial<Model['GqlModel']> & { id: string })[],
            updateInput: Model['GqlUpdate'][],
            preMap: { [key in GqlModelType]?: any }, // Result of pre function on every model included in the mutation, by model type
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
        onCreated?: Model['GqlCreate'] extends Record<string, any> ? ({ created, prisma, userData }: {
            authData: { [id: string]: { [x: string]: any } },
            created: (RecursivePartial<Model['GqlModel']> & { id: string })[],
            preMap: { [key in GqlModelType]?: any }, // Result of pre function on every model included in the mutation, by model type
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void> : never,
        onUpdated?: Model['GqlUpdate'] extends Record<string, any> ? ({ updated, updateInput, prisma, userData }: {
            authData: { [id: string]: { [x: string]: any } },
            updated: (RecursivePartial<Model['GqlModel']> & { id: string })[],
            updateInput: Model['GqlUpdate'][],
            preMap: { [key in GqlModelType]?: any }, // Result of pre function on every model included in the mutation, by model type
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void> : never,
        /**
         * Triggered before an object is deleted. This is useful if you need to find data about 
         * the deleting objects' relationships, which may be cascaded on delete
         */
        beforeDeleted?: ({ deletingIds, prisma, userData }: {
            deletingIds: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<any>,
        onDeleted?: ({ beforeDeletedData, deleted, deletedIds, prisma, userData }: {
            beforeDeletedData: any,
            deleted: Count,
            deletedIds: string[],
            preMap: { [key in GqlModelType]?: any }, // Result of pre function on every model included in the mutation, by model type
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    }
    yup: {
        create?: (params: any) => (Model['GqlCreate'] extends Record<string, any> ? ObjectSchema<any, any, any, any> : any),
        update?: (params: any) => (Model['GqlUpdate'] extends Record<string, any> ? ObjectSchema<any, any, any, any> : any),
    }
}

/**
 * Functions for displaying an object
 */
export type Displayer<
    Model extends {
        PrismaSelect: ModelLogicType['PrismaSelect'],
        PrismaModel: ModelLogicType['PrismaModel'],
    }
> = {
    /**
     * Select query for object's label
     */
    select: () => Model['PrismaSelect'],
    /**
     * Uses labelSelect to get label for object
     */
    label: (select: Model['PrismaModel'], languages: string[]) => string,
}

/**
 * Mapper for associating a model's many-to-many relationship names with
 * their corresponding join table names.
 */
export type JoinMap = { [key: string]: string };