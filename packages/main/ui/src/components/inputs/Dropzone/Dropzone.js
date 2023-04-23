import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Button, Grid } from "@mui/material";
import { makeStyles } from "@mui/styles";
import { useEffect, useState } from "react";
import { useDropzone } from "react-dropzone";
import { PubSub } from "../../../utils/pubsub";
const useStyles = makeStyles((theme) => ({
    gridPad: {
        paddingLeft: theme.spacing(1),
        paddingRight: theme.spacing(1),
    },
    itemPad: {
        marginTop: theme.spacing(1),
        marginBottom: theme.spacing(1),
    },
    dropContainer: {
        background: "white",
        color: "black",
        border: "3px dashed gray",
        borderRadius: "5px",
    },
    thumbsContainer: {
        display: "flex",
        flexDirection: "row",
        flexWrap: "wrap",
        marginTop: 16,
    },
    thumb: {
        display: "inline-flex",
        borderRadius: 2,
        border: "1px solid #eaeaea",
        marginBottom: 8,
        marginRight: 8,
        width: 100,
        height: 100,
        padding: 4,
        boxSizing: "border-box",
    },
    thumbInner: {
        display: "flex",
        minWidth: 0,
        overflow: "hidden",
    },
    img: {
        display: "block",
        width: "auto",
        height: "100%",
    },
}));
export const Dropzone = ({ acceptedFileTypes = ["image/*", ".heic", ".heif"], dropzoneText = "Drag 'n' drop files here or click", onUpload, showThumbs = true, maxFiles = 100, uploadText = "Upload file(s)", cancelText = "Cancel upload", disabled = false, }) => {
    const classes = useStyles();
    const [files, setFiles] = useState([]);
    const { getRootProps, getInputProps } = useDropzone({
        accept: acceptedFileTypes,
        maxFiles,
        onDrop: acceptedFiles => {
            if (acceptedFiles.length <= 0) {
                PubSub.get().publishSnack({ messageKey: "FilesNotAccepted", severity: "Error" });
                return;
            }
            setFiles(acceptedFiles.map(file => Object.assign(file, {
                preview: URL.createObjectURL(file),
            })));
        },
    });
    const upload = (e) => {
        e.stopPropagation();
        if (files.length === 0) {
            PubSub.get().publishSnack({ messageKey: "NoFilesSelected", severity: "Error" });
            return;
        }
        onUpload(files);
        setFiles([]);
    };
    const cancel = (e) => {
        e.stopPropagation();
        setFiles([]);
    };
    const thumbs = files.map(file => (_jsx("div", { className: classes.thumb, children: _jsx("div", { className: classes.thumbInner, children: _jsx("img", { src: file.preview, className: classes.img, alt: "Dropzone preview" }) }) }, file.name)));
    useEffect(() => () => {
        files.forEach(file => URL.revokeObjectURL(file.preview));
    }, [files]);
    return (_jsx("section", { className: classes.dropContainer, children: _jsxs("div", { style: { textAlign: "center" }, ...getRootProps({ className: "dropzone" }), children: [_jsx("input", { ...getInputProps() }), _jsx("p", { children: dropzoneText }), showThumbs &&
                    _jsx("aside", { className: classes.thumbsContainer, children: thumbs }), _jsxs(Grid, { className: classes.gridPad, container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(Button, { className: classes.itemPad, disabled: disabled || files.length === 0, fullWidth: true, onClick: upload, children: uploadText }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(Button, { className: classes.itemPad, disabled: disabled || files.length === 0, fullWidth: true, onClick: cancel, children: cancelText }) })] })] }) }));
};
//# sourceMappingURL=Dropzone.js.map