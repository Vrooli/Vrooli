/**
 * Custom Dropzone component for auto-generated forms. Typically this would 
 * include an "Upload" and "Cancel" button, and only send the file data to the 
 * parent component when the "Upload" button is clicked. Instead, this sends 
 * the file data to the parent component when the file is dropped, and displays files as tags 
 * which can be removed.
 * //TODO do what this comment says
 */
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import type { Theme } from "@mui/material";
import { makeStyles } from "@mui/styles";
import { useEffect, useState } from "react";
import { useDropzone } from "react-dropzone";
import { PubSub } from "../../../utils/pubsub.js";
import { type DropzoneProps } from "../types.js";

export const MAX_DROPZONE_FILES = 100;

const useStyles = makeStyles((theme: Theme) => ({
    gridPad: {
        paddingLeft: theme.spacing(1),
        paddingRight: theme.spacing(1),
    },
    itemPad: {
        marginTop: theme.spacing(1),
        marginBottom: theme.spacing(1),
    },
    dropContainer: {
        background: theme.palette.background.paper,
        color: theme.palette.background.textPrimary,
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

export function Dropzone({
    acceptedFileTypes = ["image/*", ".heic", ".heif"],
    dropzoneText = "Drag 'n' drop files here or click",
    onUpload,
    showThumbs = true,
    maxFiles = MAX_DROPZONE_FILES,
    uploadText = "Upload file(s)",
    cancelText = "Cancel upload",
    disabled = false,
}: DropzoneProps) {
    const classes = useStyles();
    const [files, setFiles] = useState<any[]>([]);
    const { getRootProps, getInputProps } = useDropzone({
        accept: acceptedFileTypes.length > 0 ? acceptedFileTypes : undefined,
        maxFiles,
        onDrop: acceptedFiles => {
            if (acceptedFiles.length <= 0) {
                PubSub.get().publish("snack", { messageKey: "FilesNotAccepted", severity: "Error" });
                return;
            }
            setFiles(acceptedFiles.map(file => Object.assign(file, {
                preview: URL.createObjectURL(file),
            })));
        },
    });

    function upload(e) {
        if (disabled) return;
        e.stopPropagation();
        if (files.length === 0) {
            PubSub.get().publish("snack", { messageKey: "NoFilesSelected", severity: "Error" });
            return;
        }
        onUpload(files);
        setFiles([]);
    }

    function cancel(e) {
        e.stopPropagation();
        setFiles([]);
    }

    const thumbs = files.map((file, index) => (
        <div className={classes.thumb} key={`${file.name}-${index}`} data-testid={`dropzone-thumb-${file.name}`}>
            <div className={classes.thumbInner}>
                <img
                    src={file.preview}
                    className={classes.img}
                    alt="Dropzone preview"
                    data-testid={`dropzone-thumb-img-${file.name}`}
                />
            </div>
        </div>
    ));

    useEffect(() => () => {
        // Make sure to revoke the data uris to avoid memory leaks
        files.forEach(file => URL.revokeObjectURL(file.preview));
    }, [files]);

    return (
        <section className={classes.dropContainer} data-testid="dropzone-container">
            <div style={{ textAlign: "center" }} {...getRootProps({ className: "dropzone" })} data-testid="dropzone-area">
                <input {...getInputProps()} data-testid="dropzone-input" />
                <p data-testid="dropzone-text">{dropzoneText}</p>
                {showThumbs &&
                    <aside className={classes.thumbsContainer} data-testid="dropzone-thumbs">
                        {thumbs}
                    </aside>}
                <Grid className={classes.gridPad} container spacing={2}>
                    <Grid item xs={12} sm={6}>
                        <Button
                            className={classes.itemPad}
                            disabled={disabled || files.length === 0}
                            fullWidth
                            onClick={upload}
                            variant="contained"
                            data-testid="dropzone-upload-button"
                        >{uploadText}</Button>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <Button
                            className={classes.itemPad}
                            disabled={files.length === 0}
                            fullWidth
                            onClick={cancel}
                            variant="outlined"
                            data-testid="dropzone-cancel-button"
                        >{cancelText}</Button>
                    </Grid>
                </Grid>
            </div>
        </section>
    );
}
