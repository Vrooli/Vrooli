import { Stack, Typography } from "@mui/material";
import { Dropzone } from "components/inputs/Dropzone/Dropzone";
import { DropzoneProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedDropzone = ({
    disabled,
    fieldData,
    index,
    onUpload,
}: GeneratedInputComponentProps) => {
    console.log('rendering dropzone');
    const props = useMemo(() => fieldData.props as DropzoneProps, [fieldData.props]);

    return (
        <Stack direction="column" key={`field-${fieldData.fieldName}-${index}`} spacing={1}>
            <Typography variant="h5" textAlign="center">{fieldData.label}</Typography>
            <Dropzone
                acceptedFileTypes={props.acceptedFileTypes}
                disabled={disabled}
                dropzoneText={props.dropzoneText}
                uploadText={props.uploadText}
                cancelText={props.cancelText}
                maxFiles={props.maxFiles}
                showThumbs={props.showThumbs}
                onUpload={(files) => { onUpload(fieldData.fieldName, files) }}
            />
        </Stack>
    );
}