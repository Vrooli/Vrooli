import { BookmarkFor, NavigableObject, OrArray, ProjectVersion, ReactionFor, ReportFor, RoutineVersion, RunProject, RunRoutine } from "@local/shared";
import { ButtonProps } from "@mui/material";
import { FormSchema } from "forms/types";
import React from "react";
import { FormErrors, PartialWithType, SvgProps, SxType } from "types";
import { Status } from "utils/consts";
import { SearchType } from "utils/search/objectToSearch";
import { ViewDisplayType } from "views/types";

export interface AdvancedSearchButtonProps {
    advancedSearchParams: object | null;
    advancedSearchSchema: FormSchema | null | undefined;
    controlsUrl: boolean;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (params: object | null) => unknown;
}

export interface BuildEditButtonsProps {
    canSubmitMutate: boolean;
    canCancelMutate: boolean;
    errors: BottomActionsButtonsProps["errors"];
    handleCancel: () => unknown;
    handleScaleChange: (delta: number) => unknown;
    handleSubmit: () => unknown;
    isAdding: boolean;
    isEditing: boolean;
    loading: boolean;
    scale: number;
}

export type CameraButtonProps = {
    disabled?: boolean;
    onTranscriptChange: (result: string) => unknown;
}

export interface CommentsButtonProps {
    /** Defaults to 0 */
    commentsCount: number | null;
    disabled?: boolean;
    object: NavigableObject | null | undefined;
}

export interface EllipsisActionButtonProps {
    children: JSX.Element | null | undefined;
}

export interface BottomActionsGridProps {
    children: OrArray<JSX.Element | null | undefined>;
    display: ViewDisplayType
    sx?: SxType;
}

export interface BottomActionsButtonsProps {
    disabledCancel?: boolean;
    disabledSubmit?: boolean;
    display: ViewDisplayType;
    errors?: FormErrors | undefined;
    hideButtons?: boolean;
    /** Hides button text on mobile */
    hideTextOnMobile?: boolean;
    isCreate: boolean;
    loading?: boolean;
    onCancel: () => unknown;
    onSetSubmitting?: (isSubmitting: boolean) => unknown;
    onSubmit?: () => unknown;
    sideActionButtons?: OrArray<JSX.Element | null | undefined>;
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
}

export type MicrophoneButtonProps = {
    disabled?: boolean;
    onTranscriptChange: (result: string) => unknown;
}

export interface PopupMenuProps extends ButtonProps {
    text?: string;
    children: JSX.Element | null | undefined;
}

export interface ReportButtonProps {
    forId: string;
    reportFor: ReportFor;
}

export interface ReportsButtonProps {
    reportsCount: number | null; // Defaults to 0
    object: NavigableObject | null | undefined;
}

export interface ReportsLinkProps {
    object: (NavigableObject & { reportsCount?: number }) | null | undefined;
}

export interface RunButtonProps {
    canUpdate: boolean;
    handleRunAdd: (run: RunProject | RunRoutine) => unknown;
    handleRunDelete: (run: RunProject | RunRoutine) => unknown;
    isBuildGraphOpen: boolean;
    isEditing: boolean;
    runnableObject: PartialWithType<ProjectVersion | RoutineVersion> | null;
}

export type SearchButtonsProps = {
    advancedSearchParams: object | null;
    advancedSearchSchema: FormSchema | null | undefined;
    controlsUrl: boolean;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (params: object | null) => unknown;
    setSortBy: (sortBy: string) => unknown;
    setTimeFrame: (timeFrame: TimeFrame | undefined) => unknown;
    sortBy: string;
    sortByOptions: any; // No way to specify generic enum
    sx?: SxType;
    timeFrame: TimeFrame | undefined;
}

export interface ShareButtonProps {
    object: NavigableObject | null | undefined;
}

export interface SideActionsButtonsProps {
    children: OrArray<JSX.Element | null | undefined>;
    display: ViewDisplayType;
    sx?: SxType;
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
}

export interface StatusMessageArray {
    status: Status;
    messages: string[];
}

export interface StatusButtonProps extends ButtonProps {
    status: Status;
    messages: string[];
}

export type TimeFrame = {
    after?: Date;
    before?: Date;
}

export interface TimeButtonProps {
    setTimeFrame: (timeFrame: TimeFrame | undefined) => unknown;
    timeFrame: TimeFrame | undefined;
}

export interface VoteButtonProps {
    disabled?: boolean;
    score?: number; // Net score - can be negative
    emoji?: string | null; // If not passed, then there is neither an upvote nor a downvote
    objectId: string;
    voteFor: ReactionFor;
    onChange: (newEmoji: string | null, newScore: number) => unknown;
}
