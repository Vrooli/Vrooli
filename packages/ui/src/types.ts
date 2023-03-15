// Defines common props
import { RoutineStepType } from 'utils';
import { FetchResult } from "@apollo/client";
import { AwardCategory, GqlModelType, NodeLink, RoutineVersion, SearchException, Session } from '@shared/consts';
import { CommonKey } from '@shared/translations';

/**
 * Top-level props that can be passed into any routed component
 */
export interface CommonProps {
    session: Session | undefined;
}

/**
 * Data to display title information for a component, which may not
 * always be translated.
 */
export type OptionalTranslation = {
    help?: string;
    helpKey?: CommonKey;
    helpVariables?: { [x: string]: string | number };
    title?: string;
    titleKey?: CommonKey;
    titleVariables?: { [x: string]: string | number };
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
    root?: {
        __typename: `${GqlModelType}`,
        handle?: string | null,
        id: string,
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

/**
 * All information required to display an award, its progress, and information about the next tier.
 */
export type AwardDisplay = {
    category: AwardCategory;
    categoryTitle: string;
    categoryDescription: string;
    earnedTier?: {
        title: string;
        description: string;
        level: number;
    },
    nextTier?: {
        title: string;
        description: string;
        level: number;
    },
    progress: number;
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
    root?: {
        __typename: `${GqlModelType}`,
        handle?: string | null,
        id: string
    } | null;
    versions?: { id: string }[] | null;
    isFromHistory?: boolean;
    isBookmarked?: boolean;
    label: string;
    bookmarks?: number;
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

// Add isLeftHanded to MUI theme
declare module '@mui/material/styles' {
    interface Theme {
        isLeftHanded: boolean;
    }
    // allow configuration using `createTheme`
    interface ThemeOptions {
        isLeftHanded?: boolean;
    }
}

// Apollo GraphQL
export type ApolloResponse = FetchResult<any, Record<string, any>, Record<string, any>>;

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
export type MaybeLazyAsync<T> = T | (() => T) | (() => Promise<T>);