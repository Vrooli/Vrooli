import { Count, SessionUser } from "../endpoints/types";
import { PrismaType, PromiseOrValue, RecursivePartial, SingleOrArray } from "../types";
import { ArraySchema } from 'yup';
import { PartialGraphQLInfo, PartialPrismaSelect, PrismaDelegate } from "../builders/types";

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
    'View' |
    'Vote' |
    'Wallet';

/**
* Basic structure of an object's business layer.
* Every business layer object has at least a PrismaType object and a format converter. 
* Everything else is optional
*/
export type ModelLogic<
    GQLObject extends { [key: string]: any },
    SuppFields extends readonly string[],
    Create extends false | MutaterShapes,
    Update extends false | MutaterShapes,
    RelationshipCreate extends false | MutaterShapes,
    RelationshipUpdate extends false | MutaterShapes,
    SearchInput,
    SortBy extends string,
    OrderBy extends { [x: string]: any; },
    Where extends { [x: string]: any; },
    GQLCreate extends { [x: string]: any; },
    GQLUpdate extends { [x: string]: any; },
    PrismaObject extends { [x: string]: any; },
    PermissionObject extends { [x: string]: any; },
    PermissionsSelect extends { [x: string]: any; },
    OwnerOrMemberWhere extends { [x: string]: any; },
    IsTransferable extends boolean,
    IsVersioned extends boolean,
> = {
    __typename: GraphQLModelType;
    format: Formatter<GQLObject, SuppFields>;
    display: Displayer<PermissionsSelect, PrismaObject>;
    delegate: (prisma: PrismaType) => PrismaDelegate;
    search?: Searcher<SearchInput, SortBy, OrderBy, Where>;
    mutate?: Mutater<GQLObject, Create, Update, RelationshipCreate, RelationshipUpdate>;
    validate?: Validator<GQLCreate, GQLUpdate, PrismaObject, PermissionObject, PermissionsSelect, OwnerOrMemberWhere, IsTransferable, IsVersioned>;
}

/**
 * Mostly unsafe type for a model logic object.
 */
export type AniedModelLogic<GQLObject extends { [x: string]: any }> = ModelLogic<GQLObject, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any>;

/**
 * Allows Prisma select fields to map to GraphQLModelTypes. Any field which can be 
 * an object (e.g. a relation) should be able to specify either a GraphQLModelType or
 * a nested ValidateMap
 */
export type ValidateMap<T> = {
    [K in keyof T]?: GraphQLModelType | ValidateMap<T[K]>
};

/**
 * An object which can maps GraphQL fields to GraphQLModelTypes. Normal fields 
 * are a single GraphQLModelType, while unions are an array of GraphQLModelTypes
 */
export type RelationshipMap<T> = {
    [K in keyof T]?: SingleOrArray<GraphQLModelType>
}

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
    }) => [SuppFields[number], () => any][]
}

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export interface Formatter<
    GQLModel extends Record<string, any>,
    SuppFields extends readonly string[]
> {
    /**
     * Maps relationship names to their GraphQL type. 
     * If the relationship is a union (i.e. has mutliple possible types), 
     * the GraphQL type will be an object of field/GraphQLModelType pairs. 
     * NOTE: The keyword "root" is used to indicate that a relationship belongs in the root 
     * object. This only applies when working with versioned data
     */
    relationshipMap: RelationshipMap<GQLModel>;
    /**
     * Map used to add/remove join tables from the query
     */
    joinMap?: { [x: string]: string };
    /**
     * List of fields which provide the count of a relationship. 
     * These fields must end with "Count", and exist in the GraphQL object.
     * Each field of the form {relationship}Count will be converted to
     * { [relationship]: { _count: true } } in the Prisma query
     * 
     * NOTE: Only allows string keys
     */
     countFields?: (keyof GQLModel extends infer R ? R extends `${string}Count` ? R : never : never)[];
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


/**
 * Describes shape of component that can be sorted in a specific order
 */
export type Searcher<SearchInput, SortBy extends string, OrderBy extends { [x: string]: any }, Where extends { [x: string]: any }> = {
    defaultSort: SortBy;
    sortMap: Record<SortBy, OrderBy>;
    searchStringQuery: ({ insensitive, languages, searchString }: {
        insensitive: { contains: string; mode: 'default' | 'insensitive'; },
        languages?: string[],
        searchString: string,
    }) => Where;
    customQueries?: (input: SearchInput, userData: SessionUser | null | undefined) => Where;
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
 * Describes shape of component that has validation rules 
 */
export type Validator<
    GQLCreate extends { [x: string]: any },
    GQLUpdate extends { [x: string]: any },
    PrismaObject extends { [x: string]: any },
    PermissionObject extends { [x: string]: any },
    PermissionsSelect extends { [x: string]: any },
    OwnerOrMemberWhere extends { [x: string]: any },
    IsTransferable extends boolean,
    IsVersioned extends boolean,
> = {
    /**
     * Maps relationsips on the object's database schema to the corresponding GraphQL type,
     * if they require validation
     * 
     * Examples include: 
     * routine -> organization
     * node -> routine -> organization
     * createdByOrganizationId
     * 
     * Examples when this is not needed:
     * project -> resourceList
     */
    validateMap: { __typename: GraphQLModelType } & ValidateMap<PermissionsSelect>;
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
    permissionsSelect: (userId: string | null, languages: string[]) => PermissionsSelect;
    /**
     * Array of resolvers to calculate the object's permissions
     */
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }: {
        isAdmin: boolean,
        isDeleted: boolean,
        isPublic: boolean,
    }) => [keyof Omit<PermissionObject, '__typename'>, () => any][];
    /**
     * Partial queries for various visibility checks
     */
    visibility: {
        /**
         * For private objects (i.e. only the owner can see them)
         */
        private: OwnerOrMemberWhere;
        /**
         * For public objects (i.e. anyone can see them)
         */
        public: OwnerOrMemberWhere;
        /**
         * For both private and public objects that you own
         */
        owner: (userId: string) => OwnerOrMemberWhere;
    }
    /**
     * Uses query result to determine if the object is soft-deleted
     */
    isDeleted: (data: PrismaObject, languages: string[]) => boolean;
    /**
     * Uses query result to determine if the object is public. This typically means "isPrivate" and "isDeleted" are false
     */
    isPublic: (data: PrismaObject, languages: string[]) => boolean;
    /**
     * Permissions data for the object's owner
     */
    owner: (data: PrismaObject) => {
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
        connect?: ({ connectMany, languages, prisma, userData }: {
            connectMany: string[],
            languages: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void;
        create?: ({ createMany, deltaAdding, languages, prisma, userData }: {
            createMany: GQLCreate[],
            deltaAdding: number,
            languages: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void;
        delete?: ({ deleteMany, languages, prisma, userData }: {
            deleteMany: string[],
            languages: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void;
        disconnect?: ({ disconnectMany, languages, prisma, userData }: {
            disconnectMany: string[],
            languages: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void;
        update?: ({ languages, prisma, updateMany, userData }: {
            languages: string[],
            prisma: PrismaType,
            updateMany: GQLUpdate[],
            userData: SessionUser,
        }) => Promise<void> | void;
    };
    /**
     * Any custom transformations you want to perform before a create/update mutation, 
     * besides the ones supported by default in cudHelper. This includes converting creates to 
     * connects, which means this function has to be pretty flexible in what it allows
     */
    transformations?: {
        create?: (createMany: GQLCreate[], userId: string) => Promise<GQLCreate[]> | GQLCreate[];
        update?: (updateMany: GQLUpdate[], userId: string) => Promise<GQLUpdate[]> | GQLUpdate[];
    };
    /**
     * True if you are allowed to tranfer the object to another user/orgaization
     */
    isTransferable: IsTransferable;
} & (
        IsTransferable extends true ? {
            /*
            * Determines if object has its original owner
            */
            hasOriginalOwner: (data: PrismaObject) => boolean;
        } : {}
    ) & (
        IsVersioned extends true ? {
            /**
             * Determines if there is a completed version of the object
             */
            hasCompletedVersion: (data: PrismaObject) => boolean;
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

export type MutaterShapes = {
    graphql: { [x: string]: any },
    db: { [x: string]: any },
}

export type RelBuilderInput<RelName extends string, Relationship extends MutaterShapes> = {
    data: { [x in RelName]: Relationship['graphql'] },
    prisma: PrismaType,
    relationshipName: RelName,
    userData: SessionUser,
}

/**
 * Describes shape of component that can be mutated
 */
export type Mutater<
    GQLObject extends { [x: string]: any },
    Create extends MutaterShapes | false,
    Update extends MutaterShapes | false,
    RelationshipCreate extends MutaterShapes | false,
    RelationshipUpdate extends MutaterShapes | false,
> = {
    /**
     * Shapes data for create/update mutations, both as a main 
     * object and as a relationship object
     */
    shape: (Create extends MutaterShapes ? {
        create: ({ data, prisma, userData }: {
            data: Create['graphql'],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<Create['db']>,
    } : {}) & (Update extends MutaterShapes ? {
        update: ({ data, prisma, userData, where }: {
            data: Update['graphql'],
            prisma: PrismaType,
            userData: SessionUser,
            where: { id: string },
        }) => PromiseOrValue<Update['db']>,
    } : {}) & (RelationshipCreate extends MutaterShapes ? {
        relCreate: ({ data, prisma, userData }: {
            data: RelationshipCreate['graphql'],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<RelationshipCreate['db']>,
    } : {}) & (RelationshipUpdate extends MutaterShapes ? {
        relUpdate: ({ data, prisma, userData }: {
            data: RelationshipUpdate['graphql'],
            prisma: PrismaType,
            userData: SessionUser,
            where: { id: string },
        }) => PromiseOrValue<RelationshipUpdate['db']>,
    } : {}),
    /**
     * Triggers when a mutation is performed on the object
     */
    trigger?: (Create extends MutaterShapes ? {
        onCreated?: ({ created, prisma, userData }: {
            authData: { [id: string]: { [x: string]: any } },
            created: (RecursivePartial<GQLObject> & { id: string })[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    } : {}) & (Update extends MutaterShapes ? {
        onUpdated?: ({ updated, updateInput, prisma, userData }: {
            authData: { [id: string]: { [x: string]: any } },
            updated: (RecursivePartial<GQLObject> & { id: string })[],
            updateInput: Update['graphql'][],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    } : {}) & {
        onDeleted?: ({ deleted, prisma, userData }: {
            deleted: Count,
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    },
    yup: (Create extends false ? {} : {
        create: ArraySchema<any, any, any, any>,
    }) & (Update extends false ? {} : {
        update: ArraySchema<any, any, any, any>,
    });
}

/**
 * Functions for displaying an object
 */
export type Displayer<
    PrismaSelect extends { [x: string]: any },
    PrismaSelectData extends { [x: string]: any }
> = {
    /**
     * Select query for object's label
     */
    select: () => PrismaSelect,
    /**
     * Uses labelSelect to get label for object
     */
    label: (select: PrismaSelectData, languages: string[]) => string,
}

/**
 * Mapper for associating a model's many-to-many relationship names with
 * their corresponding join table names.
 */
export type JoinMap = { [key: string]: string };