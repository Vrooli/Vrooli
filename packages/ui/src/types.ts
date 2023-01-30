// Defines common props
import { RoutineStepType } from 'utils';
import { FetchResult } from "@apollo/client";
import { Path } from '@shared/route/src/useLocation';
import { TFuncKey } from 'i18next';
import { GqlModelType, NodeLink, RoutineVersion, SearchException, Session } from '@shared/consts';

// Top-level props that can be passed into any routed component
export type SessionChecked = boolean;
export interface CommonProps {
    session: Session;
    sessionChecked: SessionChecked;
}

/**
 * Wraps an object with a field
 */
export type Wrap<T, K extends string> = { [P in K]: T };

/**
 * Wrapper for GraphQL input types
 */
export type IWrap<T> = { input: T }


/**
 * An object connected to routing
 */
export type NavigableObject = {
    __typename: `${GqlModelType}` | 'Shortcut' | 'Action',
    handle?: string | null,
    id: string,
    projectVersion?: {
        __typename: 'ProjectVersion',
        id: string
    } | null,
    routineVersion?: {
        __typename: 'RoutineVersion',
        id: string
    } | null,
    to?: {
        __typename: `${GqlModelType}`,
        handle?: string | null,
        id: string,
    }
}

// Common query input groups
export type IsCompleteInput = {
    isComplete?: boolean;
    isCompleteExceptions?: SearchException[];
}
export type IsInternalInput = {
    isInternal?: boolean;
    isInternalExceptions?: SearchException[];
}

/**
 * Converts objects as represented in the UI (especially forms) to create/update 
 * input objects for the GraphQL API.
 */
export type ShapeModel<
    T extends {},
    TCreate extends {} | null,
    TUpdate extends {} | null
> = (TCreate extends null ? {} : { create: (item: T) => TCreate }) &
    (TUpdate extends null ? {} : {
        update: (o: T, u: T, assertHasUpdate?: boolean) => TUpdate | undefined,
        hasObjectChanged?: (o: T, u: T) => boolean,
    }) & { idField?: keyof T & string }

// Routine-related props
export interface BaseStep {
    name: string, // name from node
    description: string | null, // description from node
}
export interface DecisionStep extends BaseStep {
    links: NodeLink[]
    type: RoutineStepType.Decision,
}
export interface SubroutineStep extends BaseStep {
    index: number,
    routineVersion: RoutineVersion
    type: RoutineStepType.Subroutine,
}
export interface RoutineListStep extends BaseStep {
    /**
     * Node's ID if object was created from a node
     */
    nodeId?: string | null,
    /**
     * Subroutine's ID if object was created from a subroutine
     */
    routineVersionId?: string | null,
    isOrdered: boolean,
    type: RoutineStepType.RoutineList,
    steps: RoutineStep[],
}
export type RoutineStep = DecisionStep | SubroutineStep | RoutineListStep

export interface ObjectOption {
    __typename: `${GqlModelType}`;
    handle?: string | null;
    id: string;
    root?: { id: string } | null;
    versions?: { id: string }[] | null;
    isFromHistory?: boolean;
    isStarred?: boolean;
    label: string;
    stars?: number;
    [key: string]: any;
    runnableObject?: {
        __typename: `${GqlModelType}`
        id: string
    } | null,
    to?: {
        __typename: `${GqlModelType}`,
        handle?: string | null,
        id: string,
    }
}

export interface ShortcutOption {
    __typename: 'Shortcut';
    isFromHistory?: boolean;
    label: string;
    id: string; // Actually URL, but id makes it easier to use
}

export interface ActionOption {
    __typename: 'Action';
    canPerform: (session: Session) => boolean;
    id: string;
    isFromHistory?: boolean;
    label: string;
}

export type AutocompleteOption = ObjectOption | ShortcutOption | ActionOption;

export type VersionInfo = {
    versionIndex: number;
    versionLabel: string;
    versionNotes?: string | null;
}

declare global {
    // Enable Nami integration
    interface Window { cardano: any; }
    // Add ID to EventTarget
    interface EventTarget { id?: string; }
}
// Enable Nami integration
window.cardano = window.cardano || {};

// Apollo GraphQL
export type ApolloResponse = FetchResult<any, Record<string, any>, Record<string, any>>;
export type ApolloError = {
    message?: string;
    graphQLErrors?: {
        message: string;
        extensions?: {
            code: string;
        };
    }[];
}

// Translations
export type ErrorKey = TFuncKey<'error', undefined>
export type CommonKey = TFuncKey<'common', undefined>

// Miscellaneous types
export type SetLocation = (to: Path, options?: { replace?: boolean }) => void;

/**
 * Makes a value nullable. Mimics the Maybe type in GraphQL.
 */
export type Maybe<T> = T | null;

/**
 * Recursively removes the Maybe type from all fields in a type, and makes them required.
 */
export type NonMaybe<T> = { [K in keyof T]-?: T[K] extends Maybe<any> ? NonNullable<T[K]> : T[K] };

/**
 * Makes a value lazy or not
 */
export type MaybeLazy<T> = T | (() => T);

/**
 * A nested Partial type, where each non-object field is a boolean.
 * Arrays are also treated as objects. Also adds:
 * - The __define field, which can be used to define fragments to include in the selection.
 * - The __union field, which can be used to define a union type (supports using fragments from the __define field)
 * - The __use field, which can be used to reference a single fragment from the __define field 
 */
export type DeepPartialBooleanWithFragments<T extends { __typename: string }> = {
    /**
     * Specifies the selection type
     */
    __selectionType?: 'common' | 'full' | 'list' | 'nav';
    /**
     * Specifies the object type
     */
    __typename?: T['__typename'];
    /**
     * Fragments to include in the selection. Each fragment's key can be used to reference it in the selection.
     */
    __define?: { [key: string]: MaybeLazy<DeepPartialBooleanWithFragments<T>> };
    /**
     * Creates a union of the specified types
     */
    __union?: { [key in T['__typename']]?: (string | number | MaybeLazy<DeepPartialBooleanWithFragments<T>>) };
    /**
     * Defines a fragment to include in the selection. The fragment can be referenced in the selection using the __use field.
     */
    __use?: string | number
} & {
        [P in keyof T]?: T[P] extends Array<infer U> ?
        U extends { __typename: string } ?
        MaybeLazy<DeepPartialBooleanWithFragments<NonMaybe<U>>> :
        boolean :
        T[P] extends { __typename: string } ?
        MaybeLazy<DeepPartialBooleanWithFragments<NonMaybe<T[P]>>> :
        boolean;
    }

/**
 * Ensures that a GraphQL selection is valid for a given type.
 */
export type GqlPartial<
    T extends { __typename: string },
> = {
    /**
     * Must specify the typename, in case we need to use it in a fragment.
     */
    __typename: T['__typename'];
    /**
     * Fields which are always included. This is is recursive partial of T
     */
    common?: DeepPartialBooleanWithFragments<NonMaybe<T>>;
    /**
     * Fields included in the full selection. Combined with common.
     */
    full?: DeepPartialBooleanWithFragments<NonMaybe<T>>;
    /**
     * Fields included in the minimal (list) selection. Combined with common.
     * If not provided, defaults to the same as full.
     */
    list?: DeepPartialBooleanWithFragments<NonMaybe<T>>;
    /**
     * Fields included to get the name and navigation info for an object.
     * NOT combined with common.
     */
    nav?: DeepPartialBooleanWithFragments<NonMaybe<T>>;

}