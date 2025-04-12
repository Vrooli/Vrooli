import { DragDropContext, Draggable, DropResult, Droppable } from "@hello-pangea/dnd";
import { CheckboxFormInput, CheckboxFormInputProps, getFormikFieldName, updateArray } from "@local/shared";
import { Box, Button, Checkbox, Divider, FormControl, FormControlLabel, FormGroup, FormHelperText, IconButton, TextField, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { randomString } from "../../../utils/codes.js";
import { IntegerInputBase } from "../IntegerInput/IntegerInput.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "./styles.js";
import { FormInputProps } from "./types.js";

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
const ADD_OPTION_INPUT_HEIGHT = 56;
const addOptionStyle = {
    marginLeft: "-7px",
    textTransform: "none",
    justifyContent: "flex-start",
} as const;
const textInputWithSideButtonStyle = {
    "& .MuiInputBase-root": {
        borderRadius: "5px 0 0 5px",
    },
} as const;
const dragIconBoxStyle = {
    cursor: "move",
    display: "flex",
    alignItems: "center",
    paddingLeft: "8px",
} as const;
const closeIconBoxStyle = {
    padding: "4px",
    paddingLeft: "8px",
} as const;
const editingOptionControlLabelStyle = {
    "& .MuiFormControlLabel-label": {
        display: "flex",
    },
} as const;
const fieldNameInputStyle = { marginBottom: "10px" } as const;

function AddCustomValueElement({
    customOptionInput,
    setCustomOptionInput,
    handleAddCustomOption,
    disabled,
    addCustomValueTextFieldRef,
    palette,
    t,
}) {
    function onInputChange(event) {
        setCustomOptionInput(event.target.value);
    }

    function onInputKeyDown(event) {
        if (event.key === "Enter") {
            event.preventDefault();
            handleAddCustomOption();
        }
    }

    const buttonStyle = {
        borderRadius: "0 5px 5px 0",
        background: palette.secondary.main,
        color: palette.secondary.contrastText,
        height: `${addCustomValueTextFieldRef.current?.clientHeight ?? ADD_OPTION_INPUT_HEIGHT}px`,
    } as const;

    return (
        <Box marginTop={1} display="flex">
            <TextField
                label="Add custom option"
                value={customOptionInput}
                onChange={onInputChange}
                onKeyDown={onInputKeyDown}
                disabled={disabled}
                inputRef={addCustomValueTextFieldRef}
                sx={textInputWithSideButtonStyle}
            />
            <Button
                onClick={handleAddCustomOption}
                disabled={disabled}
                sx={buttonStyle}
            >
                {t("Add")}
            </Button>
        </Box>
    );
}

export function FormInputCheckbox({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<CheckboxFormInput>) {
    const { palette, typography } = useTheme();
    const { t } = useTranslation();

    const [field, meta, helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));
    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [showMore, setShowMore] = useState(false);
    const [editingOptionIndex, setEditingOptionIndex] = useState<number | null>(null);
    const [editedOptionLabel, setEditedOptionLabel] = useState("");

    const addCustomValueTextFieldRef = useRef<HTMLDivElement | null>(null);
    const [customOptions, setCustomOptions] = useState<string[]>([]);
    const [customOptionInput, setCustomOptionInput] = useState("");

    const handleChange = useCallback((index: number, checked: boolean, isCustom = false) => {
        const totalOptionsLength = props.options.length + customOptions.length;
        const currentValue = Array.isArray(field.value) ? [...field.value] : [];
        // Ensure currentValue has the correct length
        while (currentValue.length < totalOptionsLength) {
            currentValue.push(false);
        }
        const newValue = updateArray(currentValue, index, checked);
        helpers.setValue(newValue);

        if (isEditing && !isCustom) {
            const defaultValueArray = (props.defaultValue ?? []).slice();
            while (defaultValueArray.length < props.options.length) {
                defaultValueArray.push(false);
            }
            const newDefaultValue = updateArray(defaultValueArray, index, checked);
            const newProps = { ...props, defaultValue: newDefaultValue };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [field.value, helpers, isEditing, props, onConfigUpdate, fieldData, customOptions.length]);

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

    const handleAddCustomOption = useCallback(() => {
        if (customOptionInput.trim() === "") return;

        if (props.maxCustomValues && customOptions.length >= props.maxCustomValues) {
            // Exceeded max custom values
            return;
        }

        setCustomOptions([...customOptions, customOptionInput.trim()]);
        setCustomOptionInput("");
    }, [customOptionInput, customOptions, props.maxCustomValues]);
    const handleRemoveCustomOption = useCallback((index: number) => {
        const newCustomOptions = customOptions.filter((_, i) => i !== index);
        setCustomOptions(newCustomOptions);

        // Also update the formik value
        const customOptionIndex = props.options.length + index;
        const newValue = updateArray(Array.isArray(field.value) ? field.value : [], customOptionIndex, false);
        helpers.setValue(newValue);
    }, [customOptions, props.options.length, field.value, helpers]);

    const totalOptions = useMemo(function totalOptionsMemo() {
        return [...props.options, ...customOptions.map((label) => ({ label, value: `custom-${label}` }))];
    }, [props.options, customOptions]);

    const maxSelectionsReached = useMemo(function maxSelectionsReachedMemo() {
        const selectedCount = (field.value || []).filter(Boolean).length;
        return props.maxSelection !== undefined && selectedCount >= props.maxSelection;
    }, [field.value, props.maxSelection]);

    useEffect(() => {
        // Ensure the field value array is the correct length
        const totalOptionsLength = totalOptions.length;
        if (Array.isArray(field.value) && field.value.length !== totalOptionsLength) {
            const newValue = Array(totalOptionsLength).fill(false);
            for (let i = 0; i < field.value.length && i < totalOptionsLength; i++) {
                newValue[i] = field.value[i];
            }
            helpers.setValue(newValue);
        }
    }, [totalOptions.length, field.value, helpers]);

    const FormControlElement = useCallback(({ children }: { children: React.ReactNode }) => {
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
                {children}
                {meta.touched && !!meta.error && (
                    <FormHelperText>
                        {typeof meta.error === "string" ? meta.error : JSON.stringify(meta.error)}
                    </FormHelperText>
                )}
            </FormControl>
        );
    }, [disabled, fieldData.id, fieldData.fieldName, fieldData.isRequired, meta.error, meta.touched]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    if (!isEditing) {
        return (
            <FormControlElement>
                <FormGroup
                    aria-labelledby={`label-${fieldData.fieldName}`}
                    row={props.row === true}
                >
                    {totalOptions.map((option, index) => {
                        const isCustomOption = index >= props.options.length;
                        function onCheckboxChange(_, checked: boolean) {
                            handleChange(index, checked, isCustomOption);
                        }

                        return (
                            <FormControlLabel
                                key={option.value}
                                control={
                                    <Checkbox
                                        checked={field.value?.[index] === true}
                                        color="secondary"
                                        onChange={onCheckboxChange}
                                        name={`${fieldData.fieldName}-${index}`}
                                        id={`${fieldData.fieldName}-${index}`}
                                        value={option.value}
                                        disabled={disabled || (props.maxSelection !== undefined && !field.value?.[index] && maxSelectionsReached)}
                                    />
                                }
                                label={
                                    <Box display="flex" alignItems="center">
                                        <Typography variant="body1">
                                            {option.label}
                                        </Typography>
                                        {!isEditing && isCustomOption && (
                                            <IconButton
                                                onClick={() => handleRemoveCustomOption(index - props.options.length)}
                                                size="small"
                                                sx={{ marginLeft: "4px" }}
                                            >
                                                <IconCommon
                                                    decorative
                                                    fill={palette.background.textSecondary}
                                                    name="Close"
                                                    size={16}
                                                />
                                            </IconButton>
                                        )}
                                    </Box>
                                }
                            />
                        );
                    })}
                    {props.allowCustomValues && (
                        <AddCustomValueElement
                            disabled={disabled || (props.maxCustomValues !== undefined && customOptions.length >= props.maxCustomValues)}
                            customOptionInput={customOptionInput}
                            setCustomOptionInput={setCustomOptionInput}
                            handleAddCustomOption={handleAddCustomOption}
                            addCustomValueTextFieldRef={addCustomValueTextFieldRef}
                            palette={palette}
                            t={t}
                        />
                    )}
                </FormGroup>
            </FormControlElement>
        );
    }
    return (
        <div>
            <FormControlElement>
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
                                    function onLabelPress(event: React.MouseEvent<HTMLDivElement>) {
                                        if (!isEditing) return;
                                        event.preventDefault();
                                        startEditingOption(index);
                                    }
                                    function onTextFieldChange(event: React.ChangeEvent<HTMLInputElement>) {
                                        setEditedOptionLabel(event.target.value);
                                    }
                                    function onRemoveOption() {
                                        removeOption(index);
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
                                                                color="secondary"
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
                                                                        onChange={onTextFieldChange}
                                                                        onKeyDown={handleOptionLabelKeyDown}
                                                                        size="small"
                                                                        value={editedOptionLabel}
                                                                        sx={optionTextFieldStyle}
                                                                    />
                                                                ) : (
                                                                    <Typography
                                                                        onClick={onLabelPress}
                                                                        style={{ cursor: isEditing ? "pointer" : "default" }}
                                                                        variant="body1"
                                                                    >
                                                                        {option.label}
                                                                    </Typography>
                                                                )}
                                                                {isEditing && (
                                                                    <>
                                                                        <Tooltip title="Drag to reorder" placement={props.row === true ? "top" : "right"}>
                                                                            <div {...providedDrag.dragHandleProps} style={dragIconBoxStyle}>
                                                                                <IconCommon
                                                                                    decorative
                                                                                    fill={palette.background.textSecondary}
                                                                                    name="Drag"
                                                                                    size={16}
                                                                                />
                                                                            </div>
                                                                        </Tooltip>
                                                                        <Tooltip title="Remove option" placement={props.row === true ? "top" : "right"}>
                                                                            <IconButton onClick={onRemoveOption} sx={closeIconBoxStyle}>
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
                                                        sx={editingOptionControlLabelStyle}
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
                            </FormGroup>
                        )}
                    </Droppable>
                </DragDropContext>
                {props.allowCustomValues && (
                    <AddCustomValueElement
                        disabled={true}
                        customOptionInput={customOptionInput}
                        setCustomOptionInput={setCustomOptionInput}
                        handleAddCustomOption={handleAddCustomOption}
                        addCustomValueTextFieldRef={addCustomValueTextFieldRef}
                        palette={palette}
                        t={t}
                    />
                )}
            </FormControlElement>
            {isEditing && props.defaultValue?.some(Boolean) && <DefaultWarningLabel variant="caption">
                Checked options will be selected by default when the form is used.
            </DefaultWarningLabel>}
            <Divider />
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
                        style={fieldNameInputStyle}
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
                    <FormControlLabel
                        control={
                            <Checkbox
                                checked={props.allowCustomValues === true}
                                color="secondary"
                                onChange={(e) => updateProp({ allowCustomValues: e.target.checked })}
                                name="allowCustomValues"
                            />
                        }
                        label="Allow custom values"
                    />
                    {props.allowCustomValues && (
                        <IntegerInputBase
                            fullWidth
                            label={"Max custom values"}
                            min={1}
                            name="maxCustomValues"
                            onChange={(value) => { updateProp({ maxCustomValues: value }); }}
                            tooltip="The maximum number of custom values a user can add."
                            value={props.maxCustomValues ?? 0}
                            zeroText="No limit"
                        />
                    )}
                </FormSettingsSection>
            )}
        </div>
    );
}
