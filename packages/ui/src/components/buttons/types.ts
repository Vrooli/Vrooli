/* c8 ignore start */
// AI_CHECK: TYPE_SAFETY=replaced-object-any-types | LAST: 2025-06-28
import type { ButtonProps } from "@mui/material";
import { type BookmarkFor, type FormSchema, type NavigableObject, type OrArray, type ProjectVersion, type ReactionFor, type ReportFor, type RoutineVersion, type SearchType, type Status, type TimeFrame } from "@vrooli/shared";
import type React from "react";
import { type FormErrors, type SxType, type ViewDisplayType } from "../../types.js";

export type AutoFillButtonProps = {
    handleAutoFill: () => unknown;
    isAutoFillLoading: boolean;
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
    display: ViewDisplayType | `${ViewDisplayType}`;
    sx?: SxType;
}

export interface BottomActionsButtonsProps {
    disabledCancel?: boolean;
    disabledSubmit?: boolean;
    display: ViewDisplayType | `${ViewDisplayType}`;
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

export interface HelpButtonProps extends Omit<ButtonProps, "size"> {
    id?: string;
    /** Markdown displayed in the popup menu */
    markdown: string;
    /** Handler to change markdown. If provided, allows editing */
    onMarkdownChange?: (markdown: string) => unknown;
    /** On click event. Not needed to open the menu */
    onClick?: (event: React.MouseEvent) => unknown;
    /** Size of the button */
    size?: number;
    /** Style applied to the root element */
    sxRoot?: SxType;
}

export type MicrophoneButtonProps = {
    disabled?: boolean;
    fill?: string;
    height?: number;
    onTranscriptChange: (result: string) => unknown;
    showWhenUnavailable?: boolean;
    width?: number;
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
    isEditing: boolean;
    objectType: "ProjectVersion" | "RoutineVersion";
    runnableObject: ProjectVersion | RoutineVersion | null;
}

export type PermissionsFilter = "All" | "Own" | "Public" | "Team";

export type SearchButtonsProps = {
    advancedSearchParams: Record<string, unknown> | null;
    advancedSearchSchema: FormSchema | null | undefined;
    controlsUrl: boolean;
    permissionsFilter?: PermissionsFilter;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (params: Record<string, unknown> | null) => unknown;
    setPermissionsFilter?: (filter: PermissionsFilter) => unknown;
    setSortBy: (sortBy: string) => unknown;
    setTimeFrame: (timeFrame: TimeFrame | undefined) => unknown;
    sortBy: string;
    sortByOptions: Record<string, string>; // Maps enum values to display strings
    sx?: SxType;
    timeFrame: TimeFrame | undefined;
}

export interface ShareButtonProps {
    object: NavigableObject | null | undefined;
}

export interface SideActionsButtonsProps {
    children: OrArray<JSX.Element | null | undefined>;
    display: ViewDisplayType | `${ViewDisplayType}`;
    sx?: SxType;
}

export interface BookmarkButtonProps {
    disabled?: boolean;
    isBookmarked?: boolean | null; // Defaults to false
    objectId: string;
    onChange?: (isBookmarked: boolean, event?: React.MouseEvent) => unknown;
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

export interface VoteButtonProps {
    disabled?: boolean;
    score?: number; // Net score - can be negative
    emoji?: string | null; // If not passed, then there is neither an upvote nor a downvote
    objectId: string;
    voteFor: ReactionFor;
    onChange: (newEmoji: string | null, newScore: number) => unknown;
}
