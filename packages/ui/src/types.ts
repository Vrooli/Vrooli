// Defines common props
import { AwardCategory, CommonKey, NodeLink } from "@local/shared";
import { Theme } from "@mui/material";
import { SystemStyleObject } from "@mui/system";
import { RunStepType } from "utils/consts";
import { RunnableRoutineVersion } from "views/runs/types";

/** 
 * An object which at least includes its type.
 * Useful when the object has not been fully loaded
 */
export type PartialWithType<T extends { __typename: string }> = Partial<T> & { __typename: T["__typename"] };
export type PartialArrayWithType<T extends { __typename: string }[]> = Array<PartialWithType<T[number]>>;
export type PartialOrArrayWithType<T> = T extends { __typename: string }[] ?
    PartialArrayWithType<T> :
    T extends { __typename: string } ?
    PartialWithType<T> :
    never;
export interface SvgProps {
    fill?: string;
    iconTitle?: string;
    id?: string;
    style?: {
        [key: string]: string | number | null,
    } & {
        [key: `&:${string}`]: { [key: string]: string | number | null },
        [key: `@media ${string}`]: { [key: string]: string | number | null },
    };
    onClick?: () => unknown;
    width?: number | string | null;
    height?: number | string | null;
}

export type SvgComponent = (props: SvgProps) => JSX.Element;

export type FormErrors = { [key: string]: string | string[] | null | undefined | FormErrors | FormErrors[] };

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

/** Wraps an object with a field */
export type Wrap<T, K extends string> = { [P in K]: T };

/** Wrapper for GraphQL input types */
export type IWrap<T> = { input: T }

/** Extracts the arguments from a function */
export type ArgsType<T> = T extends (...args: infer U) => any ? U : never;

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

/**
 * Converts objects as represented in the UI (especially forms) to create/update 
 * input objects for the GraphQL API.
 */
export type ShapeModel<
    T extends object,
    TCreate extends object | null,
    TUpdate extends object | null
> = (TCreate extends null ? object : { create: (item: T) => TCreate }) &
    (TUpdate extends null ? object : {
        update: (o: T, u: T, assertHasUpdate?: boolean) => TUpdate | undefined,
        hasObjectChanged?: (o: T, u: T) => boolean,
    }) & { idField?: keyof T & string }



/** Basic information provided to all routine steps */
export interface BaseStep {
    /** The step's name, taken from its node if relevant */
    name: string,
    /** The step's description, taken from its node if relevant */
    description: string | null,
    /** 
     * The step's location in the run, as a list of natural numbers. Examples:
     * - Root step: []
     * - First node in MultiRoutineStep root: [1]
     * - Second node in MultiRoutineStep root: [2]
     * - Third node in a RoutineListStep belonging to a MultiRoutineStep root: [3, 1]
     */
    location: number[],
}
/** Step information for all nodes in a routine */
export interface NodeStep extends BaseStep {
    /** Location of the next node to run */
    nextLocation: number[] | null,
    /** The ID of this node */
    nodeId: string,
}
/**
 * Implicit step (i.e. created based on how nodes are linked, rather 
 * than being a node itself) that represents a decision point in a routine.
 */
export interface DecisionStep extends BaseStep {
    __type: RunStepType.Decision,
    /** The options to pick */
    options: {
        link: NodeLink,
        step: DecisionStep | EndStep | RoutineListStep,
    }[];
}
/**
 * Node that marks the end of a routine. No action needed from the user.
 */
export interface EndStep extends NodeStep {
    __type: RunStepType.End,
    nextLocation: null, // End of the routine, so no next location
    /** Whether this is considered a "success" or not */
    wasSuccessful: boolean,
}
/**
 * Node that marks the start of a routine. No action needed from the user.
 */
export interface StartStep extends NodeStep {
    __type: RunStepType.Start,
}
/**
 * Either a leaf node (i.e. does not contain any subroutines/substeps of its own) 
 * or a step that needs further querying/processing to build the substeps.
 * 
 * If further processing is needed, this should be replaced with a MultiRoutineStep.
 */
export interface SingleRoutineStep extends BaseStep {
    __type: RunStepType.SingleRoutine,
    /**
     * The routine version that we'll be running in this step, or a multi-step 
     * routine that will be used to convert this step into a MultiRoutineStep
     */
    routineVersion: RunnableRoutineVersion
}
/**
 * A list of related steps in an individual routine node
 */
export interface RoutineListStep extends NodeStep {
    __type: RunStepType.RoutineList,
    /** Whether or not the steps must be run in order */
    isOrdered: boolean,
    /**
     * The ID of the routine version containing this node
     */
    parentRoutineVersionId: string,
    /**
     * The steps in this list. Leaf nodes are represented as SingleRoutineSteps,
     * while subroutines with their own steps (i.e. nodes and links) are represented
     * as MultiRoutineSteps.
     * 
     * Steps that haven't been processed yet are represented as SingleRoutineSteps
     */
    steps: (MultiRoutineStep | SingleRoutineStep)[],
}
/** Step information for a full multi-step routine */
export interface MultiRoutineStep extends BaseStep {
    __type: RunStepType.MultiRoutine,
    /** The nodes in this routine version. Unordered */
    nodes: (DecisionStep | EndStep | RoutineListStep | StartStep)[],
    /** The links in thie routine version. Unordered */
    nodeLinks: NodeLink[],
    /** The ID of this routine version */
    routineVersionId: string,
}
export interface DirectoryStep extends BaseStep {
    __type: RunStepType.Directory,
    /** ID of directory, if this is not the root directory */
    directoryId: string | null,
    hasBeenQueried: boolean,
    isOrdered: boolean,
    isRoot: boolean,
    /** ID of the project version this step is from */
    projectVersionId: string,
    steps: ProjectStep[],
}
export type ProjectStep = DirectoryStep | SingleRoutineStep | MultiRoutineStep;
/** All available step types */
export type RunStep = DecisionStep | DirectoryStep | EndStep | MultiRoutineStep | RoutineListStep | StartStep | SingleRoutineStep;
/** All step types that can be the root step */
export type RootStep = DirectoryStep | MultiRoutineStep | SingleRoutineStep;


export type CanConnect<
    RelationShape extends ({ [key in IDField]: string } & { __typename: string }),
    IDField extends string = "id",
> = RelationShape | (Pick<RelationShape, IDField | "__typename"> & { __connect?: boolean } & { [key: string]: any });

declare global {
    interface Window {
        // Enable Nami integration
        cardano: any;
        // Enable speech recognition
        SpeechRecognition: any
        webkitSpeechRecognition: any
    }
    // Add ID to EventTarget
    interface EventTarget { id?: string; }
}
// Enable Nami integration
window.cardano = window.cardano || {};

// Add isLeftHanded to MUI theme
declare module "@mui/material/styles" {
    interface Theme {
        isLeftHanded: boolean;
    }
    // allow configuration using `createTheme`
    interface ThemeOptions {
        isLeftHanded?: boolean;
    }
}

/** Makes a value nullable. Mimics the Maybe type in GraphQL. */
export type Maybe<T> = T | null;

/** Recursively removes the Maybe type from all fields in a type, and makes them required. */
export type NonMaybe<T> = { [K in keyof T]-?: T[K] extends Maybe<unknown> ? NonNullable<T[K]> : T[K] };

/** Makes a value lazy or not */
export type MaybeLazyAsync<T> = T | (() => T) | (() => Promise<T>);

export type SxType = NonNullable<SystemStyleObject<Theme>> & {
    color?: string;
}
