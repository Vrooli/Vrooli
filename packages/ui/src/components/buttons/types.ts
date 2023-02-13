import { ButtonProps, IconButtonProps } from '@mui/material';
import { ProjectVersion, ReportFor, RoutineVersion, RunProject, RunRoutine, Session, StarFor } from '@shared/consts';
import { SvgProps } from '@shared/icons';
import React from 'react';
import { NavigableObject} from 'types';
import { Status } from 'utils';

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
    session: Session;
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

export interface GridSubmitButtonsProps {
    disabledCancel?: boolean;
    disabledSubmit?: boolean;
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
    session: Session;
}

export interface PopupMenuProps extends ButtonProps {
    text?: string;
    children: any
};

export interface ReportButtonProps {
    forId: string;
    reportFor: ReportFor;
    session: Session;
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
    session: Session;
    zIndex: number;
}

export interface ShareButtonProps {
    object: NavigableObject | null | undefined;
    zIndex: number;
}

export interface StarButtonProps {
    disabled?: boolean;
    isStar?: boolean | null; // Defaults to false
    objectId: string;
    onChange?: (isStar: boolean, event?: any) => void;
    session: Session;
    showStars?: boolean; // Defaults to true. If false, the number of stars is not shown
    starFor: StarFor;
    stars?: number | null; // Defaults to 0
    sxs?: { root?: { [key: string]: any } };
    tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
}

export interface StatusMessageArray {
    status: Status;
    messages: string[];
}

export interface StatusButtonProps extends ButtonProps {
    status: Status;
    messages: string[];
}