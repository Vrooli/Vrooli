import { DragDropContext, Draggable, Droppable, type DropResult } from "@hello-pangea/dnd";
import { Button, FormControl, FormControlLabel, FormHelperText, IconButton, Radio, RadioGroup, TextField, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { getFormikFieldName, type RadioFormInput, type RadioFormInputProps } from "@vrooli/shared";
import { useField } from "formik";
import { useCallback, useMemo, useState } from "react";
import { IconCommon } from "../../../icons/Icons.js";
import { randomString } from "../../../utils/codes.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "./styles.js";
import { type FormInputProps } from "./types.js";

const DefaultWarningLabel = styled(Typography)(({ theme }) => ({
    marginLeft: "12px",
    marginBottom: "8px",
    display: "block",
    fontStyle: "italic",
    color: theme.palette.background.textSecondary,
}));

const formControlStyle = { m: "12px" } as const;
const optionTextFieldStyle = {
    flexGrow: 1,
    width: "auto",
    "& .MuiInputBase-input": {
        padding: 0,
    },
} as const;
const addOptionStyle = {
    marginLeft: "-7px",
    textTransform: "none",
} as const;

export function FormInputRadio({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<RadioFormInput>) {
    const { palette, typography } = useTheme();

    const [field, meta, helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));
    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [showMore, setShowMore] = useState(false);
    const [editingOptionIndex, setEditingOptionIndex] = useState<number | null>(null);
    const [editedOptionLabel, setEditedOptionLabel] = useState("");

    const handleChange = useCallback((value: string) => {
        const alreadyChecked = field.value === value;
        let setTo: string | null = value;
        // If already checked and either in editing more or the field is not required, uncheck
        if (alreadyChecked && (isEditing || fieldData.isRequired !== true)) {
            setTo = null;
        }
        helpers.setValue(setTo);

        if (isEditing) {
            const newProps = { ...props, defaultValue: setTo };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [field.value, isEditing, fieldData, helpers, props, onConfigUpdate]);

    const updateProp = useCallback((updatedProps: Partial<RadioFormInputProps>) => {
        if (isEditing) {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [isEditing, props, onConfigUpdate, fieldData]);

    const updateFieldData = useCallback((updatedFieldData: Partial<RadioFormInput>) => {
        if (isEditing) {
            onConfigUpdate({ ...fieldData, ...updatedFieldData });
        }
    }, [isEditing, onConfigUpdate, fieldData]);

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
        return (
            <FormControl
                key={`field-${fieldData.id}`}
                disabled={disabled}
                required={fieldData.isRequired}
                error={meta.touched && !!meta.error}
                name={fieldData.fieldName}
                component="fieldset"
                variant="standard"
                sx={formControlStyle}
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
                                {props.options.map((option, index) => {
                                    function handleLabelPress(event: React.MouseEvent<HTMLSpanElement, MouseEvent>) {
                                        if (!isEditing) return;
                                        event.preventDefault();
                                        startEditingOption(index);
                                    }

                                    return (
                                        <Draggable key={option.value} draggableId={option.value} index={index} isDragDisabled={!isEditing}>
                                            {(providedDrag) => (
                                                <div
                                                    ref={providedDrag.innerRef}
                                                    {...providedDrag.draggableProps}
                                                    style={{
                                                        ...providedDrag.draggableProps.style,
                                                        display: "flex",
                                                        alignItems: "center",
                                                        // Add some spacing between items
                                                        marginBottom: props.row ? 0 : "8px",
                                                        marginRight: props.row ? "8px" : 0,
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
                                                                        sx={optionTextFieldStyle}
                                                                    />
                                                                ) : (
                                                                    <Typography
                                                                        onClick={handleLabelPress}
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
                                                                                <IconCommon
                                                                                    decorative
                                                                                    fill={palette.background.textSecondary}
                                                                                    name="Drag"
                                                                                    size={16}
                                                                                />
                                                                            </div>
                                                                        </Tooltip>
                                                                        <Tooltip title="Remove option" placement={props.row === true ? "top" : "right"}>
                                                                            <IconButton onClick={() => removeOption(index)} sx={{ padding: "4px", paddingLeft: "8px" }}>
                                                                                <IconCommon
                                                                                    decorative
                                                                                    fill={palette.background.textSecondary}
                                                                                    name="Close"
                                                                                    size={16}
                                                                                />
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
                                    );
                                })}
                                {/* "Add Option" button when editing */}
                                {isEditing && (
                                    <Button
                                        variant="text"
                                        onClick={addOption}
                                        startIcon={<IconCommon
                                            decorative
                                            name="Add"
                                        />}
                                        style={addOptionStyle}
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
    }, [fieldData.id, fieldData.isRequired, fieldData.fieldName, disabled, meta.touched, meta.error, onDragEnd, props.row, props.defaultValue, props.options, isEditing, field.value, addOption, editingOptionIndex, typography, submitOptionLabelChange, handleOptionLabelKeyDown, editedOptionLabel, palette.background.textSecondary, handleChange, startEditingOption, removeOption]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    if (!isEditing) {
        return RadioElement;
    }
    return (
        <div>
            {RadioElement}
            {isEditing && typeof props.defaultValue === "string" && props.defaultValue.length > 0 && <DefaultWarningLabel variant="caption">
                &quot;{props.options.find((option) => option.value === props.defaultValue)?.label ?? "Option"}&quot; will be selected by default when the form is used.
            </DefaultWarningLabel>}
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateFieldData({ isRequired: !fieldData.isRequired })}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateProp({ row: !props.row })}>
                    {props.row ? "Vertical" : "Horizontal"}
                </Button>
                <Button variant="text" sx={moreButtonStyle} onClick={() => setShowMore(!showMore)}>
                    More
                </Button>
            </FormSettingsButtonRow>
            {showMore && (
                <FormSettingsSection>
                    <TextField
                        fullWidth
                        label="Field name"
                        value={fieldData.fieldName}
                        onChange={(e) => updateFieldData({ fieldName: e.target.value })}
                        style={{ marginBottom: "10px" }}
                    />
                </FormSettingsSection>
            )}
        </div>
    );
}
