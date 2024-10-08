import { CheckboxFormInput, CheckboxFormInputProps, getFormikFieldName, updateArray } from "@local/shared";
import { Button, Checkbox, FormControl, FormControlLabel, FormGroup, FormHelperText, IconButton, TextField, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { IntegerInputBase } from "components/inputs/IntegerInput/IntegerInput";
import { useField } from "formik";
import { AddIcon, CloseIcon, DragIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { DragDropContext, Draggable, DropResult, Droppable } from "react-beautiful-dnd";
import { randomString } from "utils/codes";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "../styles";
import { FormInputProps } from "../types";

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

export function FormInputCheckbox({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<CheckboxFormInput>) {
    const { palette, typography } = useTheme();

    const [field, meta, helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));
    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [showMore, setShowMore] = useState(false);
    const [editingOptionIndex, setEditingOptionIndex] = useState<number | null>(null);
    const [editedOptionLabel, setEditedOptionLabel] = useState("");

    const handleChange = useCallback((index: number, checked: boolean) => {
        const newValue = updateArray(Array.isArray(field.value) ? field.value : [], index, checked);
        helpers.setValue(newValue);

        if (isEditing) {
            const newDefaultValue = (props.defaultValue ?? []).map((value, i) => i === index ? checked : value);
            const newProps = { ...props, defaultValue: newDefaultValue };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [field.value, helpers, isEditing, props, onConfigUpdate, fieldData]);

    const updateProp = useCallback((updatedProps: Partial<CheckboxFormInputProps>) => {
        if (isEditing) {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [isEditing, props, onConfigUpdate, fieldData]);

    const updateFieldData = useCallback((updatedFieldData: Partial<CheckboxFormInput>) => {
        if (isEditing) {
            onConfigUpdate({ ...fieldData, ...updatedFieldData });
        }
    }, [onConfigUpdate, fieldData, isEditing]);

    const addOption = useCallback(() => {
        const newOptions = [...props.options, {
            label: `Option ${props.options.length + 1}`,
            value: `option-${randomString()}`,
        }];
        const newDefaultValue = [...(props.defaultValue ?? []), false];
        updateProp({ options: newOptions, defaultValue: newDefaultValue });
    }, [props.defaultValue, props.options, updateProp]);

    const removeOption = useCallback((index: number) => {
        const newOptions = props.options.filter((_, i) => i !== index);
        const newDefaultValue = props.defaultValue?.filter((_, i) => i !== index);

        let newMinSelection = props.minSelection;
        let newMaxSelection = props.maxSelection;
        // Adjust minSelection if it's greater than the new number of options
        if (newMinSelection !== undefined && newMinSelection > newOptions.length) {
            newMinSelection = newOptions.length;
        }
        // Adjust maxSelection if it's greater than the new number of options
        if (newMaxSelection !== undefined && newMaxSelection > newOptions.length) {
            newMaxSelection = newOptions.length;
        }
        // Ensure maxSelection is not less than minSelection
        if (newMaxSelection !== undefined && newMinSelection !== undefined && newMaxSelection < newMinSelection) {
            newMaxSelection = newMinSelection;
        }

        updateProp({
            options: newOptions,
            defaultValue: newDefaultValue,
            minSelection: newMinSelection,
            maxSelection: newMaxSelection,
        });
    }, [props.defaultValue, props.maxSelection, props.minSelection, props.options, updateProp]);

    const updateOptionLabel = useCallback((index: number, label: string) => {
        const newOptions = props.options.map((option, i) =>
            i === index ? { ...option, label } : option,
        );
        updateProp({ options: newOptions });
    }, [props.options, updateProp]);

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!destination || result.type !== "checkboxOption") return;

        const newOptions = Array.from(props.options);
        const [reorderedItem] = newOptions.splice(source.index, 1);
        newOptions.splice(destination.index, 0, reorderedItem);

        const newDefaultValue = props.defaultValue ? Array.from(props.defaultValue) : undefined;
        if (newDefaultValue) {
            const [reorderedDefaultValue] = newDefaultValue.splice(source.index, 1);
            newDefaultValue.splice(destination.index, 0, reorderedDefaultValue);
        }

        updateProp({ options: newOptions, defaultValue: newDefaultValue });
    }, [props.options, props.defaultValue, updateProp]);

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

    const CheckboxElement = useMemo(function checkboxElementMemo() {

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
                    <Droppable droppableId={`droppable-${fieldData.id}`} direction={props.row ? "horizontal" : "vertical"} type="checkboxOption">
                        {(providedDrop) => (
                            <FormGroup
                                {...providedDrop.droppableProps}
                                ref={providedDrop.innerRef}
                                aria-labelledby={`label-${fieldData.fieldName}`}
                                row={props.row === true}
                            >
                                {props.options.map((option, index) => {
                                    function onCheckboxChange(_, checked: boolean) {
                                        handleChange(index, checked);
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
                                                        control={
                                                            <Checkbox
                                                                checked={isEditing ? (props.defaultValue?.[index] === true) : (field.value?.[index] === true)}
                                                                onChange={onCheckboxChange}
                                                                name={`${fieldData.fieldName}-${index}`}
                                                                id={`${fieldData.fieldName}-${index}`}
                                                                value={option.value}
                                                            />
                                                        }
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
                                    );
                                })}
                                {/* "Add Option" button when editing */}
                                {isEditing && (
                                    <Button
                                        variant="text"
                                        onClick={addOption}
                                        startIcon={<AddIcon />}
                                        style={addOptionStyle}
                                    >
                                        Add Option
                                    </Button>
                                )}
                                {providedDrop.placeholder}
                            </FormGroup>
                        )}
                    </Droppable>
                </DragDropContext>
                {meta.touched && !!meta.error && <FormHelperText>{typeof meta.error === "string" ? meta.error : JSON.stringify(meta.error)}</FormHelperText>}
            </FormControl>
        );
    }, [fieldData.id, fieldData.isRequired, fieldData.fieldName, disabled, meta.touched, meta.error, onDragEnd, props.row, props.options, props.defaultValue, handleChange, isEditing, addOption, field.value, editingOptionIndex, typography, submitOptionLabelChange, handleOptionLabelKeyDown, editedOptionLabel, palette.background.textSecondary, startEditingOption, removeOption]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    if (!isEditing) {
        return CheckboxElement;
    }
    return (
        <div>
            {CheckboxElement}
            {isEditing && props.defaultValue?.some(Boolean) && <DefaultWarningLabel variant="caption">
                Checked options will be selected by default when the form is used.
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
                    <IntegerInputBase
                        fullWidth
                        label={"Min selections"}
                        max={props.options.length}
                        min={0}
                        name="minSelection"
                        onChange={(value) => { updateProp({ minSelection: value }); }}
                        tooltip="The mimumum number of options that must be selected."
                        value={props.minSelection ?? 0}
                        zeroText="No minimum"
                    />
                    <IntegerInputBase
                        fullWidth
                        label={"Max selections"}
                        max={props.options.length}
                        min={0}
                        name="maxSelection"
                        onChange={(value) => { updateProp({ maxSelection: value }); }}
                        tooltip="The maximum number of options that can be selected."
                        value={props.maxSelection ?? 0}
                        zeroText="No maximum"
                    />
                </FormSettingsSection>
            )}
        </div>
    );
}
