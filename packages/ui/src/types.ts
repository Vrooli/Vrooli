/* c8 ignore start */
// Defines common props
import type { Theme } from "@mui/material";
import { type SystemStyleObject } from "@mui/system";
import { type AITaskInfo, type AwardCategory, type ListObject, type OrArray, type TranslationKeyCommon } from "@vrooli/shared";
import { type FormikProps } from "formik";
import { type Dispatch, type SetStateAction } from "react";
import { type IconInfo } from "./icons/Icons.js";

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

export type FormErrors = { [key: string]: string | string[] | null | undefined | FormErrors | FormErrors[] };

/**
 * Data to display title information for a component, which may not
 * always be translated.
 */
export type OptionalTranslation = {
    help?: string;
    helpKey?: TranslationKeyCommon;
    helpVariables?: { [x: string]: string | number };
    title?: string;
    titleKey?: TranslationKeyCommon;
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

export type SxType = NonNullable<SystemStyleObject<Theme>> & {
    color?: string;
}

export type CrudPropsBase = {
    display: `${ViewDisplayType}`;
    isCreate: boolean;
}
export type CrudPropsPage = CrudPropsBase & ObjectViewPropsPage & {
    display: `${ViewDisplayType}`;
    onCancel?: never;
    onClose?: never;
    onCompleted?: never;
    onDeleted?: never;
}
export type CrudPropsDialog<T extends OrArray<ListObject>> = CrudPropsBase & ObjectViewPropsDialog<T> & {
    display: `${ViewDisplayType}`;
    /** Closes the view and clears cached data */
    onCancel: () => unknown;
    /** Closes the view without clearing cached data */
    onClose: () => unknown;
    /** Closes the view, clears cached data, and returns created/updated data */
    onCompleted: (data: T) => unknown;
    /** Closes the view, clears cached data, and returns deleted object (or objects if arrray) */
    onDeleted: (data: T) => unknown;
}
export type CrudPropsPartial<T extends OrArray<ListObject>> = CrudPropsBase & ObjectViewPropsPartial<T> & {
    display: `${ViewDisplayType}`;
    onCancel?: never;
    onClose?: never;
    onCompleted?: never;
    onDeleted?: never;
}
export type CrudProps<T extends OrArray<ListObject>> = CrudPropsPage | CrudPropsDialog<T> | CrudPropsPartial<T>;

/** Views can be displayed as full pages or as dialogs */
export enum ViewDisplayType {
    Dialog = "Dialog",
    Page = "Page",
    Partial = "Partial",
}
export type ViewPropsBase = {
    display: `${ViewDisplayType}`;
};
export type ViewPropsPartial = ViewPropsBase & {
    display: `${ViewDisplayType}`;
    isOpen?: never;
    onClose?: never;
}
export type ViewPropsPage = ViewPropsBase & {
    display: `${ViewDisplayType}`;
    isOpen?: never;
    onClose?: never;
}
export type ViewPropsDialog = ViewPropsBase & {
    display: `${ViewDisplayType}`;
    isOpen?: boolean;
    onClose?: () => unknown;
}
export type ViewProps = ViewPropsPartial | ViewPropsPage | ViewPropsDialog;

export type ObjectViewPropsBase = ViewProps;
export type ObjectViewPropsPage = ObjectViewPropsBase & {
    overrideObject?: never;
}
export type ObjectViewPropsDialog<T extends OrArray<ListObject>> = ObjectViewPropsBase & {
    /**  Data known about the object, which cannot be fetched from the server or cache. */
    overrideObject?: PartialOrArrayWithType<T>;
}
export type ObjectViewPropsPartial<T extends OrArray<ListObject>> = ObjectViewPropsBase & {
    /**  Data known about the object, which cannot be fetched from the server or cache. */
    overrideObject?: PartialOrArrayWithType<T>;
}
export type ObjectViewProps<T extends OrArray<ListObject>> = ObjectViewPropsPage | ObjectViewPropsDialog<T> | ObjectViewPropsPartial<T>;
export interface PageProps {
    children: JSX.Element;
    excludePageContainer?: boolean;
    mustBeLoggedIn?: boolean;
    sessionChecked: boolean;
    redirect?: string;
    sx?: SxType;
}

export type FormProps<Model extends OrArray<ListObject>, ModelShape extends OrArray<object>> = Omit<CrudProps<Model>, "isLoading"> & FormikProps<ModelShape> & {
    disabled: boolean;
    existing: ModelShape;
    handleUpdate: Dispatch<SetStateAction<ModelShape>>;
    isReadLoading: boolean;
};

export interface Dimensions {
    width: number;
    height: number;
}

export enum AITaskDisplayState {
    /** Tssk not provided to LLM */
    Disabled = "disabled",
    /** Tssk provided to LLM with other enabled tasks */
    Enabled = "enabled",
    /** LLM instructed to use this task only */
    Exclusive = "exclusive",
}
export type AITaskDisplay = AITaskInfo & {
    /**
    * What to render on the button/chip in the input toolbar.
    * We’ll reuse `AITaskInfo.label` as the default, but this allows
    * us to override it if needed.
    */
    displayName: string;
    /** Which icon to show next to the tool name in the toolbar */
    iconInfo: IconInfo;
    /** UI state for the tool picker: disabled, enabled, or exclusive */
    state: AITaskDisplayState;
    /** Arguments to pass to the tool */
    arguments: Record<string, any>;
}
