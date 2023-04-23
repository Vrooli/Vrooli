import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Stack, Typography } from "@mui/material";
import { useMemo } from "react";
import { Dropzone } from "../../Dropzone/Dropzone";
export const GeneratedDropzone = ({ disabled, fieldData, index, onUpload, }) => {
    console.log("rendering dropzone");
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    return (_jsxs(Stack, { direction: "column", spacing: 1, children: [_jsx(Typography, { variant: "h5", textAlign: "center", children: fieldData.label }), _jsx(Dropzone, { acceptedFileTypes: props.acceptedFileTypes, disabled: disabled, dropzoneText: props.dropzoneText, uploadText: props.uploadText, cancelText: props.cancelText, maxFiles: props.maxFiles, showThumbs: props.showThumbs, onUpload: (files) => { onUpload(fieldData.fieldName, files); } })] }, `field-${fieldData.fieldName}-${index}`));
};
//# sourceMappingURL=GeneratedDropzone.js.map