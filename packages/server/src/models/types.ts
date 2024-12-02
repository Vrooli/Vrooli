import { DotNotation, GqlModelType, ObjectLimit, SessionUser, YupMutateParams } from "@local/shared";
import { AnyObjectSchema } from "yup";
import { PartialGraphQLInfo, PartialPrismaSelect } from "../builders/types";
import { PromiseOrValue } from "../types";
import { SearchMap, SearchStringMap } from "../utils";
import { InputNode } from "../utils/inputNode";
import { SortMap } from "../utils/sortMap";
import { IdsCreateToConnect, InputsById, QueryAction } from "../utils/types";

type GqlObject = Record<string, any>;
type DbObject = Record<string, any>;

/** Types and flags for an object's business layer */
export type ModelLogicType = {
    __typename: `${GqlModelType}`;
    /** GraphQL input for creating the object */
    GqlCreate: GqlObject | undefined,
    /** GraphQL input for updating the object */
    GqlUpdate: GqlObject | undefined,
    /** GraphQL input for searching the object */
    GqlSearch: GqlObject | undefined,
    /** GraphQL sort options for the object */
    GqlSort: string | undefined,
    /** GraphQL type returned when querying the object */
    GqlModel: GqlObject,
    /** GraphQL type with your permissions for the object (e.g. canUpdate, canDelete) */
    GqlPermission: GqlObject,
    /** Prisma input for creating the object */
    PrismaCreate: DbObject | undefined,
    /** Prisma input for updating the object */
    PrismaUpdate: DbObject | undefined,
    /** Prisma input for selecting the object */
    PrismaSelect: DbObject,
    /** Prisma input for filtering the object */
    PrismaWhere: DbObject | undefined,
    /** Prisma type returned when querying the object */
    PrismaModel: DbObject,
    /** Flag for whether the object can be transferred to another account */
    IsTransferable?: boolean,
    /** Flag for whether the object is versioned */
    IsVersioned?: boolean,
}

/**
* Basic structure of an object's business layer. 
* Often accessed using `ModelMap`.
*/
export type ModelLogic<
    Model extends ModelLogicType,
    SuppFields extends readonly DotNotation<Model["GqlModel"]>[],
    IdField extends keyof Model["GqlModel"] = "id",
> = {
    __typename: Model["__typename"];
    /** The db table for this object, as it appears in Prisma */
    dbTable: string;
    /** The db table for the object's translations, as it appears in Prisma */
    dbTranslationTable?: string;
    /** Functions for representing the object in different formats */
    display: Displayer<Model>;
    /** Data for converting the object between GraphQL and Prisma, and related permissions */
    format: Formatter<Model>;
    /** Handles copying the object */
    duplicate?: Duplicator<any, any>;
    /** 
     * The primary key of the object. If not provided, defaults to "id".
     * 
     * NOTE: If using this for more than tags, you might want to update the logic in `cudHelper` 
     * to avoid id collisions between different object types.
     * */
    idField?: IdField;
    /** Functions and data for searching the object */
    search: Model["GqlSearch"] extends undefined ? undefined :
    Model["GqlSort"] extends undefined ? undefined :
    Model["PrismaWhere"] extends undefined ? undefined :
    Searcher<{
        GqlModel: Model["GqlModel"],
        GqlSearch: Exclude<Model["GqlSearch"], undefined>,
        GqlSort: Exclude<Model["GqlSort"], undefined>,
        PrismaWhere: Exclude<Model["PrismaWhere"], undefined>,
    }, SuppFields>;
    /** Shapes the object for different operations, and handles related triggers */
    mutate?: Mutater<{
        GqlCreate: Model["GqlCreate"],
        GqlUpdate: Model["GqlUpdate"],
        GqlModel: Exclude<Model["GqlModel"], undefined>,
        PrismaCreate: Model["PrismaCreate"],
        PrismaUpdate: Model["PrismaUpdate"],
    }>;
    /** Permissions checking  */
    validate: Validator<{
        GqlCreate: Model["GqlCreate"],
        GqlUpdate: Model["GqlUpdate"],
        PrismaModel: Exclude<Model["PrismaModel"], undefined>,
        GqlPermission: Exclude<Model["GqlPermission"], undefined>,
        PrismaSelect: Exclude<Model["PrismaSelect"], undefined>,
        PrismaWhere: Exclude<Model["PrismaWhere"], undefined>,
        IsTransferable: Exclude<Model["IsTransferable"], undefined>,
        IsVersioned: Exclude<Model["IsVersioned"], undefined>,
    }>
} & { [x: string]: any };

/**
 * An object which maps GraphQL fields to GqlModelTypes. Normal fields 
 * are a single GqlModelType, while unions are handled with an object. 
 * A union object maps Prisma relation fields to GqlModelTypes.
 */
export type GqlRelMap<
    Typename extends `${GqlModelType}`,
    GqlModel extends ModelLogicType["GqlModel"],
    PrismaModel extends ModelLogicType["PrismaModel"],
> = {
    [K in keyof GqlModel]?: `${GqlModelType}` | ({ [K2 in keyof PrismaModel]?: `${GqlModelType}` })
} & { __typename: Typename };

/**
 * Allows Prisma select fields to map to GqlModelTypes
 */
export type PrismaRelMap<Typename extends `${GqlModelType}`, T> = {
    [K in keyof T]?: `${GqlModelType}`
} & { __typename: Typename };

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
    toGraphQL: ({ ids, objects, partial, userData }: {
        ids: string[],
        languages: string[] | undefined,
        objects: ({ id: string } & DbObject)[],
        partial: PartialGraphQLInfo,
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
        __typename: `${GqlModelType}`,
        GqlCreate: ModelLogicType["GqlCreate"],
        GqlModel: ModelLogicType["GqlModel"],
        PrismaModel: ModelLogicType["PrismaModel"],
    }
> {
    /**
     * Maps GraphQL types to GqlModelType, with special handling for unions
     */
    gqlRelMap: Model extends { PrismaModel: infer P }
    ? P extends undefined
    ? GqlRelMap<Model["__typename"], Model["GqlModel"], Model["PrismaModel"]>
    : GqlRelMap<Model["__typename"], Model["GqlModel"], NonNullable<P>>
    : GqlRelMap<Model["__typename"], Model["GqlModel"], Model["PrismaModel"]>;
    /** Defines additional information for parsing union types, used for things like building mutation input trees */
    unionFields?: { [key in keyof Model["GqlModel"]]?: Record<string, never> | { connectField: keyof Model["GqlCreate"], typeField: keyof Model["GqlCreate"] } };
    /** Maps Prisma types to GqlModelType */
    prismaRelMap: PrismaRelMap<Model["__typename"], Model["PrismaModel"]>;
    /**
     * Map used to add/remove join tables from the query. 
     * Each key is a GraphQL field, and each value is the join table's relation name
     */
    joinMap?: { [key in keyof Model["GqlModel"]]?: string };
    /**
     * List of fields which provide the count of a relationship. 
     * These fields must end with "Count", and exist in the GraphQL object.
     * Each field of the form {relationship}Count will be converted to
     * { [relationship]: { _count: true } } in the Prisma query
     */
    countFields: StringArrayMap<(keyof Model["GqlModel"] extends infer R ? R extends `${string}Count` ? R : never : never)[]>;
    /** List of fields to always exclude from GraphQL results */
    hiddenFields?: string[];
    /** Add join tables which are not present in GraphQL object */
    addJoinTables?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /** Remove join tables which are not present in GraphQL object */
    removeJoinTables?: (data: GqlObject) => GqlObject;
    /** Add _count fields */
    addCountFields?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /** Remove _count fields */
    removeCountFields?: (data: GqlObject) => GqlObject;
}

type CommonSearchFields = "after" | "take" | "ids" | "offset" | "searchString" | "sortBy" | "visibility";

export type SearchStringQueryParams = {
    insensitive: { contains: string; mode: "default" | "insensitive"; },
    languages?: string[],
    searchString: string,
}

export type SearchStringQuery<Where extends DbObject> = {
    [x in keyof Where]: Where[x] extends infer R ?
    //Check array
    R extends DbObject[] ?
    ((keyof typeof SearchStringMap)[] | SearchStringQuery<Where[x]>) :
    //Check object
    R extends DbObject ?
    (keyof typeof SearchStringMap | SearchStringQuery<Where[x]>) :
    //Else
    SearchStringQuery<Where[x]> : never
} | (keyof typeof SearchStringMap);

/**
 * Describes shape of component that can be sorted in a specific order
 */
export type Searcher<
    Model extends {
        GqlModel: Exclude<ModelLogicType["GqlModel"], undefined>,
        GqlSearch: Exclude<ModelLogicType["GqlSearch"], undefined>,
        GqlSort: Exclude<ModelLogicType["GqlSort"], undefined>,
        PrismaWhere: Exclude<ModelLogicType["PrismaWhere"], undefined>,
    },
    SuppFields extends readonly DotNotation<Model["GqlModel"]>[]
> = {
    defaultSort: Model["GqlSort"];
    /**
     * Enum of all possible sort fields for this model
     * Also ensures that each field is in the SortMap object 
     * (i.e. SortMap is a superset of Model['GqlSort'])
     */
    sortBy: { [x in Model["GqlSort"]]: keyof typeof SortMap };
    /**
     * Search input fields for this model
     * Also ensures that each field is in the SearchMap object
     * (i.e. SearchMap is a superset of Model['GqlSearch'])
     * 
     * NOTE: Excludes fields which are common to all models 
     * (or have special logic), such as "take", "after", etc.
     */
    searchFields: StringArrayMap<(keyof Model["GqlSearch"] extends infer R ?
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
    searchStringQuery: () => SearchStringQuery<Model["PrismaWhere"]>;
    /**
     * Data for adding supplemental fields to the GraphQL object
     */
    supplemental?: SupplementalConverter<SuppFields>;
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
export type PermissionsMap<ModelSelect extends GqlObject> = {
    [x in keyof ModelSelect]: ModelSelect[x] | `${GqlModelType}` | [`${GqlModelType}`, string[]]
}

export type VisibilityFuncInput = {
    searchInput: { [x: string]: any },
    userId: string,
}
/**
 * Function for building partial Prisma select queries. 
 * Used to safely query only the objects you are allowed to see.
 */
export type VisibilityFunc<PrismaWhere extends Exclude<ModelLogicType["PrismaWhere"], undefined>> = ((data: VisibilityFuncInput) => PrismaWhere);

/**
 * Describes shape of component that has validation rules 
 */
export type Validator<
    Model extends {
        GqlCreate: ModelLogicType["GqlCreate"],
        GqlUpdate: ModelLogicType["GqlUpdate"],
        PrismaModel: Exclude<ModelLogicType["PrismaModel"], undefined>,
        GqlPermission: Exclude<ModelLogicType["GqlPermission"], undefined>,
        PrismaSelect: Exclude<ModelLogicType["PrismaSelect"], undefined>,
        PrismaWhere: Exclude<ModelLogicType["PrismaWhere"], undefined>,
        IsTransferable: Exclude<ModelLogicType["IsTransferable"], undefined>,
        IsVersioned: Exclude<ModelLogicType["IsVersioned"], undefined>,
    }
> = () => ({
    /**
     * The maximum number of objects that can be created by a single user/team.
     * This depends on if the owner is a user or team, if the owner 
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
    permissionsSelect: (userId: string | null, languages: string[] | undefined) => PermissionsMap<Model["PrismaSelect"]>;
    /**
     * Key/value pair of permission fields and resolvers to calculate them.
     */
    permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }: {
        data: Model["PrismaModel"],
        isAdmin: boolean,
        isDeleted: boolean,
        isLoggedIn: boolean,
        isPublic: boolean,
        userId: string | null | undefined,
    }) => { [x in keyof Omit<Model["GqlPermission"], "type">]: () => any } & { [x in Exclude<QueryAction, "Create"> as `can${x}`]: () => boolean | Promise<boolean> };
    /**
     * Partial queries for various visibility checks
     */
    visibility: {
        /** 
         * For objects you own (both private and public).
         * 
         * Set to null if this query type is not supported. This will throw an error 
         * if the query is attempted.
         */
        own: VisibilityFunc<Model["PrismaWhere"]> | null;
        /** 
         * Union of objects you own and public objects (so it should include public objects
         * you don't own).
         * 
         * Set to null if this query type is not supported. This will throw an error
         * if the query is attempted.
         */
        ownOrPublic: VisibilityFunc<Model["PrismaWhere"]> | null;
        /** 
        * For public objects you own (i.e. intersection of objects you own and private objects).
        * 
        * Set to null if this query type is not supported. This will throw an error
        * if the query is attempted.
        */
        ownPublic: VisibilityFunc<Model["PrismaWhere"]> | null;
        /** 
         * For private objects you own.
         * 
         * Set to null if this query type is not supported. This will throw an error
         * if the query is attempted.
         */
        ownPrivate: VisibilityFunc<Model["PrismaWhere"]> | null;
        /**
         * For public objects.
         * 
         * Set to null if this query type is not supported. This will throw an error
         * if the query is attempted.
         */
        public: VisibilityFunc<Model["PrismaWhere"]> | null;
        // Allow additional visibility queries
    } & { [x: string]: VisibilityFunc<Model["PrismaWhere"]> | null };
    /**
     * Uses query result to determine if the object is soft-deleted
     */
    isDeleted: (data: Model["PrismaModel"], languages: string[] | undefined) => boolean;
    /**
     * Uses query result to determine if the object is public. This typically means "isPrivate" and "isDeleted" are false. 
     * @param data The data used to determine if the object is public
     * @param getParentInfo Used to get the data of the parent object. When using data from the 
     * database, this is not needed. When using data from a create mutation, it can be useful.
     * @param languages Languages to display error messages in
     */
    isPublic: (data: Model["PrismaModel"], getParentInfo: ((id: string, typename: `${GqlModelType}`) => any | undefined), languages: string[]) => boolean;
    /**
     * Permissions data for the object's owner
     */
    owner: (data: Model["PrismaModel"] | undefined, userId: string) => {
        Team?: ({ id: string } & DbObject) | null;
        User?: ({ id: string } & DbObject) | null;
    }
    /**
     * String fields which must be checked for profanity. You don't need to 
     * include fields in a translate object, as those will be checked
     * automatically
     */
    profanityFields?: string[];
    /**
     * Any custom transformations you want to perform before a create/update mutation, 
     * besides the ones supported by default in cudHelper. This includes converting creates to 
     * connects, which means this function has to be pretty flexible in what it allows
     */
    transformations?: {
        create?: (createMany: Model["GqlCreate"][], userId: string) => Promise<Model["GqlCreate"][]> | Model["GqlCreate"][];
        update?: (updateMany: Model["GqlUpdate"][], userId: string) => Promise<Model["GqlUpdate"][]> | Model["GqlUpdate"][];
    };
    /**
     * True if you are allowed to tranfer the object to another user/orgaization
     */
    isTransferable: Model["IsTransferable"];
} & (
        Model["IsTransferable"] extends true ? {
            /* Determines if object has its original owner */
            hasOriginalOwner: (data: Model["PrismaModel"]) => boolean;
        } : object
    ) & (
        Model["IsVersioned"] extends true ? {
            /** Determines if there is a completed version of the object */
            hasCompleteVersion: (data: Model["PrismaModel"]) => boolean;
        } : object
    ))

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
    /** Data to connect to new owner */
    owner: (id: string) => {
        Team?: Partial<Data> | null
        User?: Partial<Data> | null;
    }
}

/** 
 * Result of pre function on every model included in the mutation, by model type.
 * 
 * This means you can reference pre function results by using `preMap[__typename]`.
 **/
export type PreMap = { [key in GqlModelType]?: any };

/**
 * Describes shape of component that can be mutated
 */
export type Mutater<Model extends {
    GqlCreate: ModelLogicType["GqlCreate"],
    GqlUpdate: ModelLogicType["GqlUpdate"],
    GqlModel: ModelLogicType["GqlModel"],
    PrismaCreate: ModelLogicType["PrismaCreate"],
    PrismaUpdate: ModelLogicType["PrismaUpdate"],
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
        pre?: ({ Create, Update, Delete, userData }: {
            Create: { node: InputNode, input: Model["GqlCreate"] }[],
            Update: { node: InputNode, input: (Model["GqlUpdate"] & { id: string }) }[],
            Delete: { node: InputNode, input: string }[],
            userData: SessionUser,
            inputsById: InputsById,
        }) => PromiseOrValue<object>,
        /**
         * Determines which creates should instead be connects. This is useful for
         * things like tags, where you don't want to create a new tag if one with 
         * the same name already exists.
         * 
         * @returns An array of ids to connect to, in the same order as the input array. 
         * If the id is null, then a new object should be created.
         * 
         * NOTE: We don't have to worry about it for tags, but you should keep in mind 
         * object permissions when using this function
         */
        findConnects?: ({ Create }: {
            Create: { node: InputNode, input: Model["GqlCreate"] }[],
        }) => PromiseOrValue<(string | null)[]>,
        /** Shapes data for create mutations */
        create?: Model["GqlCreate"] extends GqlObject ?
        Model["PrismaCreate"] extends DbObject ? ({ data, idsCreateToConnect, preMap, userData }: {
            data: Model["GqlCreate"],
            idsCreateToConnect: IdsCreateToConnect,
            preMap: PreMap;
            userData: SessionUser,
        }) => PromiseOrValue<Model["PrismaCreate"]> : never : never,
        /** Shapes data for update mutations */
        update?: Model["GqlUpdate"] extends GqlObject ?
        Model["PrismaUpdate"] extends DbObject ? ({ data, idsCreateToConnect, preMap, userData }: {
            data: Model["GqlUpdate"],
            idsCreateToConnect: IdsCreateToConnect,
            preMap: PreMap,
            userData: SessionUser,
        }) => PromiseOrValue<Model["PrismaUpdate"]> : never : never
    }
    /**
     * Triggers after (or before if specified) a mutation is performed on the object. Useful for sending notifications,
     * tracking awards, and updating reputation.
     */
    trigger?: {
        /**
         * Triggered before an object is deleted. This is useful if you need to find data about 
         * the deleting objects' relationships, which may be cascaded on delete. 
         * 
         * NOTE: This is only called for top-level objects, not relationships. Handle accordingly.
         */
        beforeDeleted?: ({ beforeDeletedData, deletingIds, userData }: {
            /** Result is added to this object */
            beforeDeletedData: { [key in `${GqlModelType}`]?: object },
            deletingIds: string[],
            userData: SessionUser,
        }) => PromiseOrValue<void>,
        /**
         * A trigger that includes create, update, and delete mutations. This can be used to 
         * send notifications, track awards, update reputation, etc. Try not to mutate objects here, 
         * unless you have to (e.g. adding a new version inbetween others will trigger index updates 
         * on versions not specified in the mutation)
         */
        afterMutations?: ({ createdIds, deletedIds, updatedIds, updateInputs, userData }: {
            additionalData: Record<string, any>,
            beforeDeletedData: { [key in `${GqlModelType}`]?: object },
            createdIds: string[],
            createInputs: Model["GqlCreate"][],
            deletedIds: string[],
            resultsById: { [key: string]: unknown },
            updatedIds: string[],
            updateInputs: Model["GqlUpdate"][],
            preMap: PreMap,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    }
    /** Create and update validations. Share these with UI to make forms more reliable */
    yup: {
        create?: (params: YupMutateParams) => (Model["GqlCreate"] extends GqlObject ? AnyObjectSchema : never),
        update?: (params: YupMutateParams) => (Model["GqlUpdate"] extends GqlObject ? AnyObjectSchema : never),
    }
}

/** Functions for displaying an object */
export type Displayer<
    Model extends {
        PrismaSelect: ModelLogicType["PrismaSelect"],
        PrismaModel: ModelLogicType["PrismaModel"],
    }
> = () => ({
    /** Display the object for push notifications, etc. */
    label: {
        /** Prisma selection for the label */
        select: () => Model["PrismaSelect"],
        /** Converts the selection to a string */
        get: (select: Model["PrismaModel"], languages: string[] | undefined) => string,
    }
    /** Object representation for text embedding, which is used for search */
    embed?: {
        /** Prisma selection for the embed */
        select: () => Model["PrismaSelect"],
        /** Converts the selection to a string */
        get: (select: Model["PrismaModel"], languages: string[] | undefined) => string,
    }
})

/**
 * Mapper for associating a model's many-to-many relationship names with
 * their corresponding join table names.
 */
export type JoinMap = { [key: string]: string };
