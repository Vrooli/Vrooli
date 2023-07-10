import { BookmarkFor, ProjectVersion, ReactionFor, ReportFor, RoutineVersion, RunProject, RunRoutine, SvgProps } from "@local/shared";
import { ButtonProps, IconButtonProps } from "@mui/material";
import { FormSchema } from "forms/types";
import React, { ReactNode } from "react";
import { NavigableObject, SxType } from "types";
import { Status } from "utils/consts";
import { SearchType } from "utils/search/objectToSearch";
import { ViewDisplayType } from "views/types";

export interface AdvancedSearchButtonProps {
    advancedSearchParams: object | null;
    advancedSearchSchema: FormSchema | null | undefined;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (params: object | null) => unknown;
    zIndex: number;
}

export interface BuildEditButtonsProps {
    canSubmitMutate: boolean;
    canCancelMutate: boolean;
    errors: GridSubmitButtonsProps["errors"];
    handleCancel: () => unknown;
    handleScaleChange: (delta: number) => unknown;
    handleSubmit: () => unknown;
    isAdding: boolean;
    isEditing: boolean;
    loading: boolean;
    scale: number;
    zIndex: number;
}

export type CameraButtonProps = {
    disabled?: boolean;
    onTranscriptChange: (result: string) => unknown;
}

export interface ColorIconButtonProps extends IconButtonProps {
    background: string;
    children: ReactNode;
    disabled?: boolean;
    href?: string;
    onClick?: (event: React.MouseEvent<HTMLElement>) => unknown;
    sx?: SxType;
}

export interface CommentsButtonProps {
    /** Defaults to 0 */
    commentsCount: number | null;
    disabled?: boolean;
    object: NavigableObject | null | undefined;
}

export interface EllipsisActionButtonProps {
    children: ReactNode;
}

export interface GridActionButtonsProps {
    children: ReactNode;
    display: ViewDisplayType;
}

export interface GridSubmitButtonsProps {
    disabledCancel?: boolean;
    disabledSubmit?: boolean;
    display: ViewDisplayType;
    errors?: { [key: string]: string | string[] | null | undefined };
    isCreate: boolean;
    loading?: boolean;
    onCancel: () => unknown;
    onSetSubmitting?: (isSubmitting: boolean) => unknown;
    onSubmit?: () => unknown;
    sideActionButtons?: Omit<SideActionButtonsProps, "hasGridActions">;
    zIndex: number;
}

export interface HelpButtonProps extends ButtonProps {
    id?: string;
    /** Markdown displayed in the popup menu */
    markdown: string;
    /** On click event. Not needed to open the menu */
    onClick?: (event: React.MouseEvent) => unknown;
    /** Style applied to the root element */
    sxRoot?: object;
    /** Style applied to the question mark icon */
    sx?: SvgProps;
    zIndex: number;
}

export type MicrophoneButtonProps = {
    disabled?: boolean;
    onTranscriptChange: (result: string) => unknown;
    zIndex: number;
}

export interface PopupMenuProps extends ButtonProps {
    text?: string;
    children: ReactNode;
}

export interface ReportButtonProps {
    forId: string;
    reportFor: ReportFor;
    zIndex: number;
}

export interface ReportsButtonProps {
    reportsCount: number | null; // Defaults to 0
    object: NavigableObject | null | undefined;
}

export interface ReportsLinkProps {
    object: (NavigableObject & { reportsCount: number }) | null | undefined;
}

export interface RunButtonProps {
    canUpdate: boolean;
    handleRunAdd: (run: RunProject | RunRoutine) => unknown;
    handleRunDelete: (run: RunProject | RunRoutine) => unknown;
    isBuildGraphOpen: boolean;
    isEditing: boolean;
    runnableObject: ProjectVersion | RoutineVersion | null;
    zIndex: number;
}

export interface SearchButtonsProps {
    advancedSearchParams: object | null;
    advancedSearchSchema: FormSchema | null | undefined;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (params: object | null) => unknown;
    setSortBy: (sortBy: string) => unknown;
    setTimeFrame: (timeFrame: TimeFrame | undefined) => unknown;
    sortBy: string;
    sortByOptions: any; // No way to specify generic enum
    timeFrame: TimeFrame | undefined;
    zIndex: number;
}

export interface ShareButtonProps {
    object: NavigableObject | null | undefined;
    zIndex: number;
}

export interface SideActionButtonsProps {
    children: ReactNode;
    display: ViewDisplayType;
    /** If true, displays higher up */
    hasGridActions?: boolean;
    isLeftHanded?: boolean;
    sx?: SxType;
    zIndex: number;
}

export interface SortButtonProps {
    options: any; // No way to specify generic enum
    setSortBy: (sortBy: string) => unknown;
    sortBy: string;
}

export interface BookmarkButtonProps {
    disabled?: boolean;
    isBookmarked?: boolean | null; // Defaults to false
    objectId: string;
    onChange?: (isBookmarked: boolean, event?: any) => unknown;
    showBookmarks?: boolean; // Defaults to true. If false, the number of bookmarks is not shown
    bookmarkFor: BookmarkFor;
    bookmarks?: number | null; // Defaults to 0
    sxs?: { root?: SxType };
    zIndex: number;
}

export interface StatusMessageArray {
    status: Status;
    messages: string[];
}

export interface StatusButtonProps extends ButtonProps {
    status: Status;
    messages: string[];
    zIndex: number;
}

export type TimeFrame = {
    after?: Date;
    before?: Date;
}

export interface TimeButtonProps {
    setTimeFrame: (timeFrame: TimeFrame | undefined) => unknown;
    timeFrame: TimeFrame | undefined;
    zIndex: number;
}

export interface VoteButtonProps {
    direction?: "row" | "column";
    disabled?: boolean;
    score?: number; // Net score - can be negative
    emoji?: string | null; // If not passed, then there is neither an upvote nor a downvote
    objectId: string;
    voteFor: ReactionFor;
    onChange: (newEmoji: string | null, newScore: number) => unknown;
}
