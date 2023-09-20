import { Count, DotNotation, GqlModelType, ObjectLimit } from "@local/shared";
import { ObjectSchema } from "yup";
import { PartialGraphQLInfo, PartialPrismaSelect, PrismaDelegate } from "../builders/types";
import { PrismaType, PromiseOrValue, RecursivePartial, SessionUserToken } from "../types";
import { SearchMap, SearchStringMap } from "../utils";
import { SortMap } from "../utils/sortMap";
import { InputNode, InputsById, QueryAction } from "../utils/types";

/** Types and flags for an object's business layer */
export type ModelLogicType = {
    __typename: `${GqlModelType}`;
    /** GraphQL input for creating the object */
    GqlCreate: Record<string, any> | undefined,
    /** GraphQL input for updating the object */
    GqlUpdate: Record<string, any> | undefined,
    /** GraphQL input for searching the object */
    GqlSearch: Record<string, any> | undefined,
    /** GraphQL sort options for the object */
    GqlSort: string | undefined,
    /** GraphQL type returned when querying the object */
    GqlModel: Record<string, any>,
    /** GraphQL type with your permissions for the object (e.g. canUpdate, canDelete) */
    GqlPermission: Record<string, any>,
    /** Prisma input for creating the object */
    PrismaCreate: Record<string, any> | undefined,
    /** Prisma input for updating the object */
    PrismaUpdate: Record<string, any> | undefined,
    /** Prisma input for selecting the object */
    PrismaSelect: Record<string, any>,
    /** Prisma input for filtering the object */
    PrismaWhere: Record<string, any> | undefined,
    /** Prisma type returned when querying the object */
    PrismaModel: Record<string, any>,
    /** Flag for whether the object can be transferred to another account */
    IsTransferable?: boolean,
    /** Flag for whether the object is versioned */
    IsVersioned?: boolean,
}

/**
* Basic structure of an object's business layer. 
* Properties are often accessed using `getLogic()`.
*/
export type ModelLogic<
    Model extends ModelLogicType,
    SuppFields extends readonly DotNotation<Model["GqlModel"]>[],
    IdField extends keyof Model["GqlModel"] = "id",
> = {
    __typename: Model["__typename"];
    /** The prisma delegate for this object (e.g. prisma.user) */
    delegate: (prisma: PrismaType) => PrismaDelegate;
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
    toGraphQL: ({ ids, objects, partial, prisma, userData }: {
        ids: string[],
        languages: string[],
        objects: ({ id: string } & { [x: string]: any })[], // TODO: fix this type
        partial: PartialGraphQLInfo,
        prisma: PrismaType,
        userData: SessionUserToken | null,
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
    removeJoinTables?: (data: { [x: string]: any }) => any;
    /** Add _count fields */
    addCountFields?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /** Remove _count fields */
    removeCountFields?: (data: { [x: string]: any }) => any;
}

type CommonSearchFields = "after" | "take" | "ids" | "searchString" | "visibility";

export type SearchStringQueryParams = {
    insensitive: { contains: string; mode: "default" | "insensitive"; },
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
     * Any additional data to add to the Prisma query. Not usually needed
     */
    customQueryData?: (input: Model["GqlSearch"], userData: SessionUserToken | null) => Model["PrismaWhere"];
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
export type PermissionsMap<ModelSelect extends { [x: string]: any }> = {
    [x in keyof ModelSelect]: ModelSelect[x] | `${GqlModelType}` | [`${GqlModelType}`, string[]]
}

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
    permissionsSelect: (userId: string | null, languages: string[]) => PermissionsMap<Model["PrismaSelect"]>;
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
        /** For private objects (i.e. only the owner can see them) */
        private: Model["PrismaWhere"];
        /** For public objects (i.e. anyone can see them) */
        public: Model["PrismaWhere"];
        /** For both private and public objects that you own */
        owner: (userId: string) => Model["PrismaWhere"];
    }
    /**
     * Uses query result to determine if the object is soft-deleted
     */
    isDeleted: (data: Model["PrismaModel"], languages: string[]) => boolean;
    /**
     * Uses query result to determine if the object is public. This typically means "isPrivate" and "isDeleted" are false
     */
    isPublic: (data: Model["PrismaModel"], languages: string[]) => boolean;
    /**
     * Permissions data for the object's owner
     */
    owner: (data: Model["PrismaModel"], userId: string) => {
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
    /** Data to connect to new owner */
    owner: (id: string) => {
        Organization?: Partial<Data> | null
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
        pre?: ({ Create, Update, Delete, prisma, userData }: {
            Create: { node: InputNode, input: Model["GqlCreate"] }[],
            Update: { node: InputNode, input: (Model["GqlUpdate"] & { id: string }) }[],
            Delete: { node: InputNode, input: string }[],
            prisma: PrismaType,
            userData: SessionUserToken,
            inputsById: InputsById,
        }) => PromiseOrValue<object>,
        /** Shapes data for create mutations */
        create?: Model["GqlCreate"] extends Record<string, any> ?
        Model["PrismaCreate"] extends Record<string, any> ? ({ data, preMap, prisma, userData }: {
            data: Model["GqlCreate"],
            preMap: PreMap;
            prisma: PrismaType,
            userData: SessionUserToken,
        }) => PromiseOrValue<Model["PrismaCreate"]> : never : never,
        /** Shapes data for update mutations */
        update?: Model["GqlUpdate"] extends Record<string, any> ?
        Model["PrismaUpdate"] extends Record<string, any> ? ({ data, preMap, prisma, userData }: {
            data: Model["GqlUpdate"],
            preMap: PreMap,
            prisma: PrismaType,
            userData: SessionUserToken,
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
        beforeDeleted?: ({ beforeDeletedData, deletingIds, prisma, userData }: {
            /** Result is added to this object */
            beforeDeletedData: { [key in `${GqlModelType}`]?: object },
            deletingIds: string[],
            prisma: PrismaType,
            userData: SessionUserToken,
        }) => PromiseOrValue<void>,
        /**
         * A trigger that includes create, update, and delete mutations. This can be used to 
         * send notifications, track awards, update reputation, etc. Try not to mutate objects here, 
         * unless you have to (e.g. adding a new version inbetween others will trigger index updates 
         * on versions not specified in the mutation)
         */
        afterMutations?: ({ created, deleted, prisma, updated, updateInputs, userData }: {
            beforeDeletedData: { [key in `${GqlModelType}`]?: object },
            created: (RecursivePartial<Model["GqlModel"]> & { id: string })[],
            deleted: Count,
            deletedIds: string[],
            updated: (RecursivePartial<Model["GqlModel"]> & { id: string })[],
            updateInputs: Model["GqlUpdate"][],
            preMap: PreMap,
            prisma: PrismaType,
            userData: SessionUserToken,
        }) => PromiseOrValue<void>,
    }
    /** Create and update validations. Share these with UI to make forms more reliable */
    yup: {
        create?: (params: any) => (Model["GqlCreate"] extends Record<string, any> ? ObjectSchema<any, any, any, any> : any),
        update?: (params: any) => (Model["GqlUpdate"] extends Record<string, any> ? ObjectSchema<any, any, any, any> : any),
    }
}

/**
 * Functions for displaying an object
 */
export type Displayer<
    Model extends {
        PrismaSelect: ModelLogicType["PrismaSelect"],
        PrismaModel: ModelLogicType["PrismaModel"],
    }
> = {
    /** Display the object for push notifications, etc. */
    label: {
        /** Prisma selection for the label */
        select: () => Model["PrismaSelect"],
        /** Converts the selection to a string */
        get: (select: Model["PrismaModel"], languages: string[]) => string,
    }
    /** Object representation for text embedding, which is used for search */
    embed?: {
        /** Prisma selection for the embed */
        select: () => Model["PrismaSelect"],
        /** Converts the selection to a string */
        get: (select: Model["PrismaModel"], languages: string[]) => string,
    }
}

/**
 * Mapper for associating a model's many-to-many relationship names with
 * their corresponding join table names.
 */
export type JoinMap = { [key: string]: string };
