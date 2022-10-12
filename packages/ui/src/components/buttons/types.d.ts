import { ButtonProps } from '@mui/material';
import { ReportFor, StarFor } from '@shared/consts';
import { SvgProps } from 'assets/img/types';
import React from 'react';
import { NavigableObject, Session } from 'types';
import { ObjectType, Status } from 'utils';

export interface CommentsButtonProps {
    commentsCount: number | null; // Defaults to 0
    disabled?: boolean;
    object: { id: string, handle?: string | null, __typename: string } | null;
    tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
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
    object: { id: string, __typename: string } | null;
    tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
}

export interface ReportsLinkProps {
    object: (NavigableObject & { reportsCount: number }) | null | undefined;
}

export interface ShareButtonProps {
    objectType: ObjectType;
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