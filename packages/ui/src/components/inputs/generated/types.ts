import { Theme } from '@mui/material';
import { Session } from '@shared/consts';
import { FieldData, GridContainer, GridContainerBase } from 'forms/types';

export interface GeneratedGridProps {
    childContainers?: GridContainer[];
    fields: FieldData[];
    formik: any;
    layout?: GridContainer | GridContainerBase;
    onUpload: (fieldName: string, files: string[]) => void;
    session: Session | undefined;
    theme: Theme;
    zIndex: number;
}

export interface GeneratedInputComponentProps {
    disabled?: boolean;
    fieldData: FieldData;
    formik: any;
    index?: number;
    onUpload: (fieldName: string, files: string[]) => void;
    session: Session | undefined;
    zIndex: number;
}

export interface GeneratedInputComponentWithLabelProps extends GeneratedInputComponentProps {
    copyInput?: (fieldName: string) => void;
    textPrimary: string;
}