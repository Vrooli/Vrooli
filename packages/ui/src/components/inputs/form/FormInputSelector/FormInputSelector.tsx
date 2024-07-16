import { Button, IconButton, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { useField } from "formik";
import { SelectorFormInput, SelectorFormInputOption, SelectorFormInputProps } from "forms/types";
import { AddIcon, CloseIcon, DragIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { DragDropContext, Draggable, DropResult, Droppable } from "react-beautiful-dnd";
import { randomString } from "utils/codes";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "../styles";
import { FormInputProps } from "../types";

//TODO need to finish standard input component and item connect input component so that we can define the shape of the object being selected, or select an existing standard to use
//TODO when editing, should be able to unselect an option so that it's not marked as default. Also add "clear" button next to default value indicator
export function FormInputSelector<T extends SelectorFormInputOption>({
    disabled,
    fieldData,
    isEditing,
    onConfigUpdate,
}: FormInputProps<SelectorFormInput<T>>) {
    const { palette } = useTheme();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [field, , helpers] = useField(fieldData.fieldName);
    const handleChange = useCallback((selected: T | null) => {
        const newValue = selected !== null ? props.getOptionValue(selected) ?? undefined : undefined;
        if (isEditing) {
            const newProps = { ...props, defaultValue: newValue };
            // @ts-ignore TODO
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(newValue);
    }, [props, isEditing, helpers, onConfigUpdate, fieldData]);

    // States for showing additional customization options
    const [showOptions, setShowOptions] = useState(false);
    const [showMore, setShowMore] = useState(false);
    function toggleShowOptions() {
        setShowOptions(showOptions => !showOptions);
        setShowMore(false);
    }
    function toggleShowMore() {
        setShowMore(showMore => !showMore);
        setShowOptions(false);
    }

    const updateProp = useCallback((updatedProps: Partial<SelectorFormInputProps<T>>) => {
        if (isEditing) {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [isEditing, props, onConfigUpdate, fieldData]);

    function updateFieldData(updatedFieldData: Partial<SelectorFormInput<T>>) {
        if (isEditing) {
            onConfigUpdate({ ...fieldData, ...updatedFieldData });
        }
    }

    const addOption = useCallback(() => {
        const newOptions = [...props.options, {
            label: `Option ${props.options.length + 1}`,
            value: `option-${randomString()}`,
        }];
        // @ts-ignore TODO
        updateProp({ options: newOptions });
    }, [props.options, updateProp]);

    const removeOption = useCallback((index: number) => {
        const newOptions = props.options.filter((_, i) => i !== index);
        // @ts-ignore TODO
        const newDefaultValue = props.defaultValue === props.options[index].value ? undefined : props.defaultValue;
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
        if (!destination || result.type !== "selectorOption") return;

        const newOptions = Array.from(props.options);
        const [reorderedItem] = newOptions.splice(source.index, 1);
        newOptions.splice(destination.index, 0, reorderedItem);

        updateProp({ options: newOptions });
    }, [props.options, updateProp]);

    const SelectorElement = useMemo(() => (
        <SelectorBase
            autoFocus={props.autoFocus}
            disabled={disabled}
            fullWidth
            getOptionLabel={props.getOptionLabel}
            inputAriaLabel={props.inputAriaLabel}
            isRequired={fieldData.isRequired}
            label={props.label}
            name={fieldData.fieldName}
            noneOption={props.noneOption}
            onChange={handleChange}
            options={props.options}
            value={isEditing ? props.options?.find(option => option.value === field.value) : field.value}
            getOptionDescription={props.getOptionDescription}
            getOptionIcon={props.getOptionIcon}
            multiple={false}
            tabIndex={props.tabIndex}
        />
    ), [props.autoFocus, props.getOptionLabel, props.inputAriaLabel, props.label, props.noneOption, props.options, props.getOptionDescription, props.getOptionIcon, props.tabIndex, disabled, fieldData.isRequired, fieldData.fieldName, handleChange, isEditing, field.value]);

    const optionsButtonStyle = useMemo(function optionsButtonStyleMemo() {
        return propButtonWithSectionStyle(showOptions);
    }, [showOptions]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    if (!isEditing) {
        return SelectorElement;
    }
    return (
        <div>
            {SelectorElement}
            {isEditing && props.defaultValue !== undefined && (
                <Typography
                    variant="caption"
                    style={{
                        marginBottom: "8px",
                        marginLeft: "8px",
                        display: "block",
                        fontStyle: "italic",
                        color: palette.background.textSecondary,
                    }}
                >
                    Default: {props.getOptionLabel(props.defaultValue)}
                </Typography>
            )}
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateFieldData({ isRequired: !fieldData.isRequired })}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={optionsButtonStyle} onClick={toggleShowOptions}>
                    Options
                </Button>
                <Button variant="text" sx={moreButtonStyle} onClick={toggleShowMore}>
                    More
                </Button>
            </FormSettingsButtonRow>
            {showOptions && (
                <FormSettingsSection>
                    <Button
                        variant="outlined"
                        onClick={() => updateProp({ noneOption: !props.noneOption })}
                        style={{ alignSelf: "flex-start" }}
                    >
                        {props.noneOption ? "Disable 'None' option" : "Enable 'None' option"}
                    </Button>
                    <DragDropContext onDragEnd={onDragEnd}>
                        <Droppable droppableId={`droppable-${fieldData.id}`} direction="vertical" type="selectorOption">
                            {(providedDrop) => (
                                <div {...providedDrop.droppableProps} ref={providedDrop.innerRef}>
                                    {props.options.map((option, index) => (
                                        <Draggable key={option.value.toString()} draggableId={option.value.toString()} index={index}>
                                            {(providedDrag) => (
                                                <div
                                                    ref={providedDrag.innerRef}
                                                    {...providedDrag.draggableProps}
                                                    style={{
                                                        ...providedDrag.draggableProps.style,
                                                        display: "flex",
                                                        alignItems: "center",
                                                        marginBottom: "8px",
                                                    }}
                                                >
                                                    <TextField
                                                        value={option.label}
                                                        onChange={(e) => updateOptionLabel(index, e.target.value)}
                                                        style={{ flexGrow: 1 }}
                                                    />
                                                    <Tooltip title="Drag to reorder" placement="top">
                                                        <div {...providedDrag.dragHandleProps} style={{ cursor: "move", display: "flex", alignItems: "center", paddingLeft: "8px" }}>
                                                            <DragIcon fill={palette.background.textSecondary} width="16px" height="16px" />
                                                        </div>
                                                    </Tooltip>
                                                    <Tooltip title="Remove option" placement="top">
                                                        <IconButton onClick={() => removeOption(index)} sx={{ padding: "4px", paddingLeft: "8px" }}>
                                                            <CloseIcon fill={palette.background.textSecondary} width="16px" height="16px" />
                                                        </IconButton>
                                                    </Tooltip>
                                                </div>
                                            )}
                                        </Draggable>
                                    ))}
                                    {providedDrop.placeholder}
                                </div>
                            )}
                        </Droppable>
                    </DragDropContext>
                    <Button
                        variant="text"
                        onClick={addOption}
                        startIcon={<AddIcon />}
                        style={{
                            alignSelf: "flex-start",
                            marginTop: "8px",
                            textTransform: "none",
                        }}
                    >
                        Add Option
                    </Button>
                </FormSettingsSection>
            )}
            {showMore && (
                <FormSettingsSection>
                    <TextInput
                        fullWidth
                        label="Label"
                        onChange={(event) => { updateProp({ label: event.target.value }); }}
                        value={props.label ?? ""}
                    />
                    <TextInput
                        fullWidth
                        label="Field name"
                        onChange={(event) => { updateFieldData({ fieldName: event.target.value }); }}
                        value={fieldData.fieldName ?? ""}
                    />
                </FormSettingsSection>
            )}
        </div>
    );
}
