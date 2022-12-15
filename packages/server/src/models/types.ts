import { Count, SessionUser } from "../endpoints/types";
import { PrismaType, PromiseOrValue, RecursivePartial, SingleOrArray } from "../types";
import { ObjectSchema } from 'yup';
import { PartialGraphQLInfo, PartialPrismaSelect, PrismaDelegate } from "../builders/types";
import { SortMap } from "../utils/sortMap";
import { SearchMap, SearchStringMap } from "../utils";

export type GraphQLModelType =
    'Api' |
    'ApiKey' |
    'ApiVersion' |
    'Comment' |
    'Copy' |
    'DevelopResult' |
    'Email' |
    'Fork' |
    'Handle' |
    'HistoryResult' |
    'Issue' |
    'Label' |
    'LearnResult' |
    'Meeting' |
    'MeetingInvite' |
    'Member' |
    'MemberInvite' |
    'Node' |
    'NodeEnd' |
    'NodeLink' |
    'NodeLinkWhen' |
    'NodeLoop' |
    'NodeLoopWhile' |
    'NodeRoutineList' |
    'NodeRoutineListItem' |
    'Note' |
    'NoteVersion' |
    'Notification' |
    'NotificationSubscription' |
    'Organization' |
    'Payment' |
    'Phone' |
    'PopularResult' |
    'Post' |
    'Premium' |
    'Project' |
    'ProjectVersion' |
    'ProjectVersionDirectory' |
    'ProjectOrRoutineSearchResult' |
    'ProjectOrOrganizationSearchResult' |
    'PullRequest' |
    'PushDevice' |
    'Question' |
    'QuestionAnswer' |
    'Quiz' |
    'QuizAttempt' |
    'QuizQuestion' |
    'QuizQuestionResponse' |
    'Reminder' |
    'ReminderItem' |
    'ReminderList' |
    'Report' |
    'ReportResponse' |
    'ReputationHistory' |
    'ResearchResult' |
    'Resource' |
    'ResourceList' |
    'Role' |
    'Routine' |
    'RoutineVersion' |
    'RoutineVersionInput' |
    'RoutineVersionOutput' |
    'RunProject' |
    'RunProjectSchedule' |
    'RunProjectStep' |
    'RunRoutine' |
    'RunRoutineInput' |
    'RunRoutineSchedule' |
    'RunRoutineStep' |
    'SmartContract' |
    'SmartContractVersion' |
    'Standard' |
    'StandardVersion' |
    'Star' |
    'StatsApi' |
    'StatsNote' |
    'StatsOrganization' |
    'StatsProject' |
    'StatsQuiz' |
    'StatsRoutine' |
    'StatsSmartContract' |
    'StatsStandard' |
    'StatsUser' |
    'Tag' |
    'Transfer' |
    'User' |
    'UserSchedule' |
    'UserScheduleFilter' |
    'View' |
    'Vote' |
    'Wallet';

/**
* Basic structure of an object's business layer.
* Every business layer object has at least a PrismaType object and a format converter. 
* Everything else is optional
*/
export type ModelLogic<
    Model extends {
        GqlCreate: Record<string, any>,
        GqlUpdate: Record<string, any>,
        GqlRelCreate?: Record<string, any>,
        GqlRelUpdate?: Record<string, any>,
        GqlSearch: Record<string, any>,
        GqlSort: string,
        GqlModel: Record<string, any>,
        GqlPermission: Record<string, any>,
        PrismaCreate?: Record<string, any>,
        PrismaUpdate?: Record<string, any>,
        PrismaRelCreate?: Record<string, any>,
        PrismaRelUpdate?: Record<string, any>,
        PrismaSelect: Record<string, any>,
        PrismaWhere: Record<string, any>,
        PrismaModel: Record<string, any>,
        IsTransferable: boolean,
        IsVersioned: boolean,
    },
    SuppFields extends readonly (keyof Model['GqlModel'] extends infer R ? R extends string ? R : never : never)[],
> = {
    __typename: GraphQLModelType;
    delegate: (prisma: PrismaType) => PrismaDelegate;
    display: Displayer<Model>;
    duplicate?: Duplicator<any, any>;
    format: Formatter<Model, SuppFields>;
    search?: Searcher<Model>;
    mutate?: Mutater<Model>;
    validate?: Validator<Model>;
}

/**
 * Mostly unsafe type for a model logic object.
 */
export type AniedModelLogic<GqlModel extends { [x: string]: any }> = ModelLogic<{
    GqlCreate: Record<string, any>,
    GqlUpdate: Record<string, any>,
    GqlRelCreate?: Record<string, any>,
    GqlRelUpdate?: Record<string, any>,
    GqlSearch: Record<string, any>,
    GqlSort: string,
    GqlModel: GqlModel
    GqlPermission: Record<string, any>,
    PrismaCreate?: Record<string, any>,
    PrismaUpdate?: Record<string, any>,
    PrismaRelCreate?: Record<string, any>,
    PrismaRelUpdate?: Record<string, any>,
    PrismaSelect: Record<string, any>,
    PrismaWhere: Record<string, any>,
    PrismaModel: Record<string, any>,
    IsTransferable: boolean,
    IsVersioned: boolean,
}, any>;

/**
 * An object which can maps GraphQL fields to GraphQLModelTypes. Normal fields 
 * are a single GraphQLModelType, while unions are handled with an object. 
 * A union object maps Prisma relation fields to GraphQLModelTypes.
 */
export type GqlRelMap<
    GQLObject extends { [x: string]: any },
    PrismaObject extends { [x: string]: any },
> = {
    [K in keyof GQLObject]?: GraphQLModelType | ({ [K2 in keyof PrismaObject]?: GraphQLModelType })
} & { __typename: GraphQLModelType };

/**
 * Allows Prisma select fields to map to GraphQLModelTypes
 */
export type PrismaRelMap<T> = {
    [K in keyof T]?: GraphQLModelType
} & { __typename: GraphQLModelType };

/**
 * Helper functions for adding and removing supplemental fields. These are fields 
 * are requested in the select query, but are either not in the main database or 
 * cannot be requested in the same query (e.g. isStarred, permissions) 
 */
export interface SupplementalConverter<
    SuppFields extends readonly string[]
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
     * An array of resolver functions, one for each calculated (supplemental) field
     */
    toGraphQL: ({ ids, objects, partial, prisma, userData }: {
        ids: string[],
        languages: string[],
        objects: ({ id: string } & { [x: string]: any })[], // TODO: fix this type
        partial: PartialGraphQLInfo,
        prisma: PrismaType,
        userData: SessionUser | null,
    }) => { [key in SuppFields[number]]: () => any };
}

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export interface Formatter<
    Model extends {
        GqlModel: Record<string, any>,
        PrismaModel: Record<string, any>,
    },
    SuppFields extends readonly (keyof Model['GqlModel'] extends infer R ? R extends string ? R : never : never)[]
> {
    /**
     * Maps GraphQL types to GraphQLModelType, with special handling for unions
     */
    gqlRelMap: GqlRelMap<Model['GqlModel'], Model['PrismaModel']>;
    /**
     * Maps Prisma types to GraphQLModelType
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
     * 
     * NOTE: Only allows string keys
     */
    countFields?: (keyof Model['GqlModel'] extends infer R ? R extends `${string}Count` ? R : never : never)[];
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
        GqlSearch: Record<string, any>,
        GqlSort: string,
        PrismaWhere: Record<string, any>,
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
     * Array of search input fields for this model
     * Also ensures that each field is in the SearchMap object
     * (i.e. SearchMap is a superset of Model['GqlSearch'])
     * 
     * NOTE: Excludes fields which are common to all models (or have special logic), such as "take", "after", 
     * "visibility", etc.
     */
    searchFields: (keyof Model['GqlSearch'] extends infer R ?
        R extends keyof typeof SearchMap ?
        R extends CommonSearchFields ? never : R
        : never
        : never)[];
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

export type ObjectLimitVisibility = number | {
    noPremium: number,
    premium: number,
}

export type ObjectLimitOwner = number | {
    public: ObjectLimitVisibility,
    private: ObjectLimitVisibility,
}

export type ObjectLimit = number | {
    User: ObjectLimitOwner,
    Organization: ObjectLimitOwner,
}

/**
 * Describes shape of select query used to check 
 * the permissions for an object. Basically, it's the 
 * Prisma select query, but where fields can be a GraphQLModelType 
 * string instead of a boolean
 */
export type PermissionsMap<ModelSelect extends { [x: string]: any }> = {
    [x in keyof ModelSelect]: ModelSelect[x] | GraphQLModelType
}

/**
 * Describes shape of component that has validation rules 
 */
export type Validator<
    Model extends {
        GqlCreate?: Record<string, any>,
        GqlUpdate?: Record<string, any>,
        PrismaModel: Record<string, any>,
        GqlPermission: Record<string, any>,
        PrismaSelect: Record<string, any>,
        PrismaWhere: Record<string, any>,
        IsTransferable: boolean,
        IsVersioned: boolean,
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
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }: {
        isAdmin: boolean,
        isDeleted: boolean,
        isPublic: boolean,
    }) => {
            [x in keyof Omit<Model['GqlPermission'], '__typename'>]: () => any
        }
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
            updateMany: Model['GqlUpdate'][],
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
            hasCompletedVersion: (data: Model['PrismaModel']) => boolean;
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
    GqlCreate?: Record<string, any>,
    GqlUpdate?: Record<string, any>,
    GqlModel: Record<string, any>,
    GqlPermission: Record<string, any>,
    PrismaCreate?: Record<string, any>,
    PrismaUpdate?: Record<string, any>,
}> = {
    /**
     * Shapes data for create/update mutations, both as a main 
     * object and as a relationship object
     */
    shape: {
        create?: Model['GqlCreate'] extends Record<string, any> ?
        Model['PrismaCreate'] extends Record<string, any> ? ({ data, prisma, userData }: {
            data: Model['GqlCreate'],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<Model['PrismaCreate']> : never : never,
        update?: Model['GqlUpdate'] extends Record<string, any> ?
        Model['PrismaUpdate'] extends Record<string, any> ? ({ data, prisma, userData, where }: {
            data: Model['GqlUpdate'],
            prisma: PrismaType,
            userData: SessionUser,
            where: { id: string },
        }) => PromiseOrValue<Model['PrismaUpdate']> : never : never
    }
    /**
     * Triggers when a mutation is performed on the object
     */
    trigger?: {
        onCreated?: Model['GqlCreate'] extends Record<string, any> ? ({ created, prisma, userData }: {
            authData: { [id: string]: { [x: string]: any } },
            created: (RecursivePartial<Model['GqlModel']> & { id: string })[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void> : never,
        onUpdated?: Model['GqlUpdate'] extends Record<string, any> ? ({ updated, updateInput, prisma, userData }: {
            authData: { [id: string]: { [x: string]: any } },
            updated: (RecursivePartial<Model['GqlModel']> & { id: string })[],
            updateInput: Model['GqlUpdate'][],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void> : never,
        onDeleted?: ({ deleted, prisma, userData }: {
            deleted: Count,
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    }
    yup: {
        create?: (Model['GqlCreate'] extends Record<string, any> ? ObjectSchema<any, any, any, any> : undefined),
        update?: (Model['GqlUpdate'] extends Record<string, any> ? ObjectSchema<any, any, any, any> : undefined),
    }
}

/**
 * Functions for displaying an object
 */
export type Displayer<
    Model extends {
        PrismaSelect: Record<string, any>,
        PrismaModel: Record<string, any>,
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