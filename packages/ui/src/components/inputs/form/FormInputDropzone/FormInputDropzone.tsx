import { DropzoneFormInput, DropzoneFormInputProps } from "@local/shared";
import { Autocomplete, Button, Chip, ListItemText, MenuItem, TextField, useTheme } from "@mui/material";
import { Dropzone, MAX_DROPZONE_FILES } from "components/inputs/Dropzone/Dropzone";
import { IntegerInputBase } from "components/inputs/IntegerInput/IntegerInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { useCallback, useMemo, useState } from "react";
import { CHIP_LIST_LIMIT } from "utils/consts";
import { PubSub } from "utils/pubsub";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "../styles";
import { FormInputProps } from "../types";

const commonFileTypes = [
    { label: "All Images", value: "image/*" },
    { label: "JPEG Image", value: ".jpeg" },
    { label: "PNG Image", value: ".png" },
    { label: "GIF Image", value: ".gif" },
    { label: "PDF Document", value: ".pdf" },
    { label: "Word Document", value: ".doc" },
    { label: "Word Document (Open XML)", value: ".docx" },
    { label: "Excel Spreadsheet", value: ".xls" },
    { label: "Excel Spreadsheet (Open XML)", value: ".xlsx" },
    { label: "Text File", value: ".txt" },
    { label: "MP3 Audio", value: ".mp3" },
    { label: "MP4 Video", value: ".mp4" },
    { label: "CSV File", value: ".csv" },
    { label: "ZIP Archive", value: ".zip" },
    { label: "RAR Archive", value: ".rar" },
    { label: "HTML Document", value: ".html" },
    { label: "JavaScript File", value: ".js" },
    { label: "JSON File", value: ".json" },
    { label: "XML File", value: ".xml" },
    { label: "SVG Image", value: ".svg" },
];

const MIN_FILE_TYPE_LENGTH = 2;
const MAX_FILE_TYPE_LENGTH = 50;

/** Removes invalid characters from file type string */
function withoutInvalidChars(str: string) {
    // Allow only alphanumeric characters, periods, slashes, and asterisks
    return str.replace(/[^a-zA-Z0-9./*]/g, "");
}

export function FormInputDropzone({
    disabled,
    fieldData,
    isEditing,
    onConfigUpdate,
}: FormInputProps<DropzoneFormInput>) {
    const { palette } = useTheme();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [showFileTypes, setShowFileTypes] = useState(false);
    const [showMore, setShowMore] = useState(false);
    function toggleShowFileTypes() {
        setShowFileTypes(showFileTypes => !showFileTypes);
        setShowMore(false);
    }
    function toggleShowMore() {
        setShowMore(showMore => !showMore);
        setShowFileTypes(false);
    }

    const updateProp = useCallback(function updatePropCallback(updatedProps: Partial<DropzoneFormInputProps>) {
        if (!isEditing) {
            return;
        }
        const newProps = { ...props, ...updatedProps };
        onConfigUpdate({ ...fieldData, props: newProps });
    }, [isEditing, onConfigUpdate, props, fieldData]);
    function updateFieldData(updatedFieldData: Partial<DropzoneFormInput>) {
        if (!isEditing) {
            return;
        }
        onConfigUpdate({ ...fieldData, ...updatedFieldData });
    }

    function onUpload(fieldName: string, files: string[]) {
        console.log("TODO: Implement onUpload", fieldName, files);
    }

    const fileTypes = useMemo(() => {
        return (props.acceptedFileTypes || []).map(type => commonFileTypes.find(ft => ft.value === type) || { label: type, value: type });
    }, [props.acceptedFileTypes]);

    const [fileTypeInputValue, setFileTypeInputValue] = useState<string>("");
    const handleFileTypeAdd = useCallback(function handleFileTypeAddCallback(newType: string) {
        const updatedFileTypes = Array.isArray(props.acceptedFileTypes) ? [...props.acceptedFileTypes] : [];
        if (!updatedFileTypes.includes(newType)) {
            updatedFileTypes.push(newType);
            updateProp({ acceptedFileTypes: updatedFileTypes });
        }
        setFileTypeInputValue("");
    }, [props.acceptedFileTypes, updateProp]);
    function handleFileTypeDelete(index: number) {
        const updatedFileTypes = Array.isArray(props.acceptedFileTypes) ? [...props.acceptedFileTypes] : [];
        updatedFileTypes.splice(index, 1);
        updateProp({ acceptedFileTypes: updatedFileTypes });
    }
    const onFileInputChange = useCallback(function onFileInputChangeCallback(change: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) {
        const sanitized = withoutInvalidChars(change.target.value);
        setFileTypeInputValue(sanitized);
    }, []);
    const onFileTypeInputKeyDown = useCallback(function onFileTypeInputKeyDownCallback(event: React.KeyboardEvent<HTMLDivElement>) {
        // Check if the user pressed ',', ';', or enter
        if (!(["Comma", "Enter"].includes(event.code) || (event.code === "Semicolon" && !event.shiftKey))) return;
        event.preventDefault();
        // Remove invalid characters
        const fileType = withoutInvalidChars(fileTypeInputValue);
        // Check if file type is valid length
        if (fileType.length < MIN_FILE_TYPE_LENGTH) {
            PubSub.get().publish("snack", { message: "File type too short", severity: "Error" });
            return;
        }
        if (fileType.length > MAX_FILE_TYPE_LENGTH) {
            PubSub.get().publish("snack", { message: "File type too long", severity: "Error" });
            return;
        }
        // Determine if file type is already selected
        const isSelected = (props.acceptedFileTypes ?? []).some(t => t === fileType);
        if (isSelected) {
            PubSub.get().publish("snack", { message: "File type already selected", severity: "Error" });
            return;
        }
        // Add file type to list
        handleFileTypeAdd(fileType);
        // Clear input
        setFileTypeInputValue("");
    }, [fileTypeInputValue, handleFileTypeAdd, props.acceptedFileTypes]);

    const InputElement = useMemo(() => (
        <Dropzone
            acceptedFileTypes={props.acceptedFileTypes}
            disabled={disabled || isEditing} // Can't upload files when editing the config
            dropzoneText={props.dropzoneText}
            uploadText={props.uploadText}
            cancelText={props.cancelText}
            maxFiles={props.maxFiles}
            showThumbs={props.showThumbs}
            onUpload={(files) => { onUpload(fieldData.fieldName, files); }}
        />
    ), [props.acceptedFileTypes, props.dropzoneText, props.uploadText, props.cancelText, props.maxFiles, props.showThumbs, disabled, isEditing, fieldData.fieldName]);

    const typesButtonStyle = useMemo(function typesButtonStyleMemo() {
        return propButtonWithSectionStyle(showFileTypes);
    }, [showFileTypes]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    if (!isEditing) {
        return InputElement;
    }
    return (
        <div>
            {InputElement}
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={() => { updateFieldData({ isRequired: !fieldData.isRequired }); }}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={typesButtonStyle} onClick={toggleShowFileTypes}>
                    File types
                </Button>
                <Button variant="text" sx={moreButtonStyle} onClick={toggleShowMore}>
                    More
                </Button>
            </FormSettingsButtonRow>
            {showFileTypes && (
                <FormSettingsSection>
                    <Autocomplete
                        id={`file-types-input-${fieldData.fieldName}`}
                        fullWidth
                        freeSolo={true}
                        getOptionKey={(option) => typeof option === "string" ? option : option.value}
                        getOptionLabel={(option) => typeof option === "string" ? option : `${option.label} (${option.value})`}
                        limitTags={CHIP_LIST_LIMIT}
                        multiple
                        options={commonFileTypes}
                        value={fileTypes}
                        defaultValue={fileTypes}
                        renderTags={(value, getTagProps) =>
                            value.map((option, index) => {
                                return (
                                    <Chip
                                        {...getTagProps({ index })}
                                        id={`file-type-chip-${option.value}`}
                                        key={option.value}
                                        variant="filled"
                                        label={option.label}
                                        onDelete={() => handleFileTypeDelete(index)}
                                        sx={{
                                            backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0",
                                            color: "white",
                                        }}
                                    />
                                );
                            })}
                        renderInput={(params) => (
                            <TextField
                                value={fileTypeInputValue}
                                onChange={onFileInputChange}
                                label={"File types"}
                                placeholder={"Enter or select a file type"}
                                InputProps={{
                                    ...params.InputProps,
                                    endAdornment: (
                                        <>
                                            {params.InputProps.endAdornment}
                                        </>
                                    ),
                                }}
                                inputProps={params.inputProps}
                                onKeyDown={onFileTypeInputKeyDown}
                                fullWidth
                                sx={{ paddingRight: 0, minWidth: "250px" }}
                            />
                        )}
                        renderOption={(props, option) => (
                            <MenuItem
                                {...props}
                                onClick={(event) => {
                                    event.preventDefault();
                                    handleFileTypeAdd(option.value);
                                }}
                                selected={commonFileTypes.some(t => t.value === option.value)}
                            >
                                <ListItemText>{`${option.label} (${option.value})`}</ListItemText>
                            </MenuItem>
                        )}
                    />
                </FormSettingsSection>
            )}
            {showMore && (
                <FormSettingsSection>
                    <TextInput
                        fullWidth
                        label="Placeholder"
                        onChange={(event) => { updateProp({ dropzoneText: event.target.value }); }}
                        value={props.dropzoneText ?? ""}
                    />
                    <TextInput
                        fullWidth
                        label="Field name"
                        onChange={(event) => { updateFieldData({ fieldName: event.target.value }); }}
                        value={fieldData.fieldName ?? ""}
                    />
                    <IntegerInputBase
                        fullWidth
                        label={"Max files"}
                        max={MAX_DROPZONE_FILES}
                        min={0}
                        name="maxFiles"
                        onChange={(value) => { updateProp({ maxFiles: value }); }}
                        tooltip="Maximum number of files that can be uploaded"
                        value={props.maxFiles ?? 0}
                        zeroText={`Default max (${MAX_DROPZONE_FILES})`}
                    />
                </FormSettingsSection>
            )}
        </div>
    );
}
