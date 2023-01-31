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