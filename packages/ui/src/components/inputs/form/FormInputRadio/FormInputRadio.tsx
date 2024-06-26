import { Button, FormControl, FormControlLabel, FormHelperText, IconButton, Radio, RadioGroup, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { RadioFormInput, RadioFormInputProps } from "forms/types";
import { AddIcon, CloseIcon, DragIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { DragDropContext, Draggable, DropResult, Droppable } from "react-beautiful-dnd";
import { randomString } from "utils/codes";
import { FormInputProps } from "../types";

const propButtonStyle = { textDecoration: "underline", textTransform: "none" } as const;

export function FormInputRadio({
    disabled,
    fieldData,
    onConfigUpdate,
}: FormInputProps<RadioFormInput>) {
    const { palette, typography } = useTheme();

    const [field, meta, helpers] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [showMore, setShowMore] = useState(false);
    const [editingOptionIndex, setEditingOptionIndex] = useState<number | null>(null);
    const [editedOptionLabel, setEditedOptionLabel] = useState("");

    const handleChange = useCallback((value: string) => {
        const alreadyChecked = field.value === value;
        let setTo: string | null = value;
        // If already checked and either in editing more or the field is not required, uncheck
        if (alreadyChecked && (typeof onConfigUpdate === "function" || fieldData.isRequired !== true)) {
            setTo = null;
        }
        helpers.setValue(setTo);

        if (typeof onConfigUpdate === "function") {
            const newProps = { ...props, defaultValue: setTo };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [field.value, helpers, onConfigUpdate, props, fieldData]);

    const updateProp = useCallback((updatedProps: Partial<RadioFormInputProps>) => {
        if (typeof onConfigUpdate === "function") {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [onConfigUpdate, props, fieldData]);

    const updateFieldData = useCallback((updatedFieldData: Partial<RadioFormInput>) => {
        if (typeof onConfigUpdate === "function") {
            onConfigUpdate({ ...fieldData, ...updatedFieldData });
        }
    }, [onConfigUpdate, fieldData]);

    const addOption = useCallback(() => {
        const newOptions = [...props.options, {
            label: `Option ${props.options.length + 1}`,
            value: `option-${randomString()}`,
        }];
        updateProp({ options: newOptions });
    }, [props.options, updateProp]);

    const removeOption = useCallback((index: number) => {
        const newOptions = props.options.filter((_, i) => i !== index);
        const newDefaultValue = props.defaultValue === props.options[index].value ? null : props.defaultValue;
        updateProp({ options: newOptions, defaultValue: newDefaultValue });
    }, [props.options, props.defaultValue, updateProp]);

    const updateOptionLabel = useCallback((index: number, label: string) => {
        const newOptions = props.options.map((option, i) =>
            i === index ? { ...option, label } : option,
        );
        updateProp({ options: newOptions });
    }, [props.options, updateProp]);

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!destination || result.type !== "radioOption") return;

        const newOptions = Array.from(props.options);
        const [reorderedItem] = newOptions.splice(source.index, 1);
        newOptions.splice(destination.index, 0, reorderedItem);

        updateProp({ options: newOptions });
    }, [props.options, updateProp]);

    const startEditingOption = useCallback((index: number) => {
        setEditingOptionIndex(index);
        setEditedOptionLabel(props.options[index].label);
    }, [props.options]);
    const submitOptionLabelChange = useCallback(() => {
        if (editingOptionIndex !== null && editedOptionLabel !== props.options[editingOptionIndex].label) {
            updateOptionLabel(editingOptionIndex, editedOptionLabel);
        }
        setEditingOptionIndex(null);
    }, [editingOptionIndex, editedOptionLabel, props.options, updateOptionLabel]);
    const handleOptionLabelKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === "Enter") {
            e.preventDefault();
            submitOptionLabelChange();
        } else if (e.key === "Escape") {
            e.preventDefault();
            setEditingOptionIndex(null);
        }
    }, [submitOptionLabelChange]);

    const RadioElement = useMemo(() => {
        const isEditing = typeof onConfigUpdate === "function";
        return (
            <FormControl
                key={`field-${fieldData.id}`}
                disabled={disabled}
                required={fieldData.isRequired}
                error={meta.touched && !!meta.error}
                name={fieldData.fieldName}
                component="fieldset"
                variant="standard"
                sx={{ m: "12px" }}
            >
                <DragDropContext onDragEnd={onDragEnd}>
                    <Droppable droppableId={`droppable-${fieldData.id}`} direction={props.row ? "horizontal" : "vertical"} type="radioOption">
                        {(providedDrop) => (
                            <RadioGroup
                                {...providedDrop.droppableProps}
                                ref={providedDrop.innerRef}
                                aria-labelledby={`label-${fieldData.fieldName}`}
                                name={fieldData.fieldName}
                                value={isEditing ? props.defaultValue : field.value}
                                row={props.row === true}
                            >
                                {props.options.map((option, index) => (
                                    <Draggable key={option.value} draggableId={option.value} index={index} isDragDisabled={!isEditing}>
                                        {(providedDrag) => (
                                            <div
                                                ref={providedDrag.innerRef}
                                                {...providedDrag.draggableProps}
                                                style={{
                                                    ...providedDrag.draggableProps.style,
                                                    display: "flex",
                                                    alignItems: "center",
                                                    marginBottom: "8px", // Add some spacing between items
                                                }}
                                            >
                                                <FormControlLabel
                                                    key={option.value}
                                                    value={option.value}
                                                    control={<Radio onClick={() => handleChange(option.value)} />}
                                                    label={(
                                                        <>
                                                            {isEditing && editingOptionIndex === index ? (
                                                                <TextField
                                                                    autoFocus
                                                                    InputProps={{ style: (typography["body1"] as object || {}) }}
                                                                    onBlur={submitOptionLabelChange}
                                                                    onChange={(e) => { setEditedOptionLabel(e.target.value); }}
                                                                    onKeyDown={handleOptionLabelKeyDown}
                                                                    size="small"
                                                                    value={editedOptionLabel}
                                                                    sx={{
                                                                        flexGrow: 1,
                                                                        width: "auto",
                                                                        "& .MuiInputBase-input": {
                                                                            padding: 0,
                                                                        },
                                                                    }}
                                                                />
                                                            ) : (
                                                                <Typography
                                                                    onClick={(e) => {
                                                                        if (isEditing) {
                                                                            e.preventDefault();
                                                                            startEditingOption(index);
                                                                        }
                                                                    }}
                                                                    style={{ cursor: isEditing ? "pointer" : "default" }}
                                                                    variant="body1"
                                                                >
                                                                    {option.label}
                                                                </Typography>
                                                            )}
                                                            {isEditing && (
                                                                <>
                                                                    <Tooltip title="Drag to reorder" placement={props.row === true ? "top" : "right"}>
                                                                        <div {...providedDrag.dragHandleProps} style={{ cursor: "move", display: "flex", alignItems: "center", paddingLeft: "8px" }}>
                                                                            <DragIcon fill={palette.background.textSecondary} width="16px" height="16px" />
                                                                        </div>
                                                                    </Tooltip>
                                                                    <Tooltip title="Remove option" placement={props.row === true ? "top" : "right"}>
                                                                        <IconButton onClick={() => removeOption(index)} sx={{ padding: "4px", paddingLeft: "8px" }}>
                                                                            <CloseIcon fill={palette.background.textSecondary} width="16px" height="16px" />
                                                                        </IconButton>
                                                                    </Tooltip>
                                                                </>
                                                            )}
                                                        </>
                                                    )}
                                                    sx={{
                                                        "& .MuiFormControlLabel-label": {
                                                            display: "flex",
                                                        },
                                                    }}
                                                />
                                            </div>
                                        )}
                                    </Draggable>
                                ))}
                                {/* "Add Option" button when editing */}
                                {isEditing && (
                                    <Button
                                        variant="text"
                                        onClick={addOption}
                                        startIcon={<AddIcon />}
                                        style={{
                                            marginLeft: "-7px",
                                            textTransform: "none",
                                        }}
                                    >
                                        Add Option
                                    </Button>
                                )}
                                {providedDrop.placeholder}
                            </RadioGroup>
                        )}
                    </Droppable>
                </DragDropContext>
                {meta.touched && !!meta.error && <FormHelperText>{typeof meta.error === "string" ? meta.error : JSON.stringify(meta.error)}</FormHelperText>}
            </FormControl>
        );
    }, [onConfigUpdate, fieldData.id, fieldData.isRequired, fieldData.fieldName, disabled, meta.touched, meta.error, onDragEnd, props.row, props.options, props.defaultValue, addOption, field.value, editingOptionIndex, typography, submitOptionLabelChange, handleOptionLabelKeyDown, editedOptionLabel, palette.background.textSecondary, handleChange, startEditingOption, removeOption]);

    if (typeof onConfigUpdate !== "function") {
        return RadioElement;
    }

    return (
        <div>
            {RadioElement}
            {typeof onConfigUpdate === "function" && typeof props.defaultValue === "string" && props.defaultValue.length > 0 && <Typography
                variant="caption"
                style={{
                    marginLeft: "12px",
                    marginBottom: "8px",
                    display: "block",
                    fontStyle: "italic",
                    color: palette.background.textSecondary,
                }}>
                &quot;{props.options.find((option) => option.value === props.defaultValue)?.label ?? "Option"}&quot; will be selected by default when the form is used.
            </Typography>}
            <div style={{ display: "flex", flexDirection: "row" }}>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateFieldData({ isRequired: !fieldData.isRequired })}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateProp({ row: !props.row })}>
                    {props.row ? "Vertical" : "Horizontal"}
                </Button>
                <Button variant="text" sx={{ ...propButtonStyle, color: showMore ? "primary.main" : undefined }} onClick={() => setShowMore(!showMore)}>
                    More
                </Button>
            </div>
            {showMore && (
                <div style={{ marginTop: "10px", padding: "8px", display: "flex", flexDirection: "column", gap: "8px" }}>
                    <TextField
                        fullWidth
                        label="Field name"
                        value={fieldData.fieldName}
                        onChange={(e) => updateFieldData({ fieldName: e.target.value })}
                        style={{ marginBottom: "10px" }}
                    />
                </div>
            )}
        </div>
    );
}
