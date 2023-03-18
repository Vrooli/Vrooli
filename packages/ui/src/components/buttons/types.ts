import { ButtonProps, IconButtonProps } from '@mui/material';
import { BookmarkFor, ProjectVersion, ReportFor, RoutineVersion, RunProject, RunRoutine, VoteFor } from '@shared/consts';
import { SvgProps } from '@shared/icons';
import { FormSchema } from 'forms/types';
import React from 'react';
import { NavigableObject } from 'types';
import { Status } from 'utils/consts';
import { SearchType } from 'utils/search/objectToSearch';
import { ViewDisplayType } from 'views/types';

export interface AdvancedSearchButtonProps {
    advancedSearchParams: object | null;
    advancedSearchSchema: FormSchema | null | undefined;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (params: object | null) => void;
    zIndex: number;
}

export interface BuildEditButtonsProps {
    canSubmitMutate: boolean;
    canCancelMutate: boolean;
    errors: GridSubmitButtonsProps['errors'];
    handleCancel: () => void;
    handleSubmit: () => void;
    isAdding: boolean;
    isEditing: boolean;
    loading: boolean;
}

export type CameraButtonProps = {
    disabled?: boolean;
    onTranscriptChange: (result: string) => void;
}

export interface ColorIconButtonProps extends IconButtonProps {
    background: string;
    children: JSX.Element | null;
    disabled?: boolean;
    href?: string;
    onClick?: (event: React.MouseEvent<any>) => void;
    sx?: { [key: string]: any };
}

export interface CommentsButtonProps {
    commentsCount: number | null; // Defaults to 0
    disabled?: boolean;
    object: NavigableObject | null | undefined;
}

export interface GridActionButtonsProps {
    children: JSX.Element | JSX.Element[];
    display: ViewDisplayType;
}

export interface GridSubmitButtonsProps {
    disabledCancel?: boolean;
    disabledSubmit?: boolean;
    display: ViewDisplayType;
    errors?: { [key: string]: string | string[] | null | undefined };
    isCreate: boolean;
    loading?: boolean;
    onCancel: () => void;
    onSetSubmitting?: (isSubmitting: boolean) => void;
    onSubmit?: () => void;
}

export interface HelpButtonProps extends ButtonProps {
    id?: string;
    /**
     * Markdown displayed in the popup menu
     */
    markdown: string;
    /**
     * On click event. Not needed to open the menu
     */
    onClick?: (event: React.MouseEvent) => void;
    /**
     * Style applied to the root element
     */
    sxRoot?: object;
    /**
     * Style applied to the question mark icon
     */
    sx?: SvgProps;
}

export type MicrophoneButtonProps = {
    disabled?: boolean;
    onTranscriptChange: (result: string) => void;
}

export interface PopupMenuProps extends ButtonProps {
    text?: string;
    children: any
};

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
    handleRunAdd: (run: RunProject | RunRoutine) => void;
    handleRunDelete: (run: RunProject | RunRoutine) => void;
    isBuildGraphOpen: boolean;
    isEditing: boolean;
    runnableObject: ProjectVersion | RoutineVersion | null;
    zIndex: number;
}

export interface SearchButtonsProps {
    advancedSearchParams: object | null;
    advancedSearchSchema: FormSchema | null | undefined;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (params: object | null) => void;
    setSortBy: (sortBy: string) => void;
    setTimeFrame: (timeFrame: TimeFrame | undefined) => void;
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
    children: JSX.Element | null | boolean | undefined | (JSX.Element | null | boolean | undefined)[];
    display: ViewDisplayType;
    isLeftHanded?: boolean;
    sx?: { [key: string]: any };
    zIndex: number;
}

export interface SortButtonProps {
    options: any; // No way to specify generic enum
    setSortBy: (sortBy: string) => void;
    sortBy: string;
}

export interface BookmarkButtonProps {
    disabled?: boolean;
    isBookmarked?: boolean | null; // Defaults to false
    objectId: string;
    onChange?: (isBookmarked: boolean, event?: any) => void;
    showBookmarks?: boolean; // Defaults to true. If false, the number of bookmarks is not shown
    bookmarkFor: BookmarkFor;
    bookmarks?: number | null; // Defaults to 0
    sxs?: { root?: { [key: string]: any } };
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
    setTimeFrame: (timeFrame: TimeFrame | undefined) => void;
    timeFrame: TimeFrame | undefined;
}

export interface VoteButtonProps {
    direction?: 'row' | 'column';
    disabled?: boolean;
    score?: number; // Net score - can be negative
    isUpvoted?: boolean | null; // If not passed, then there is neither an upvote nor a downvote
    objectId: string;
    voteFor: VoteFor;
    onChange: (isUpvote: boolean | null, newScore: number) => void;
}