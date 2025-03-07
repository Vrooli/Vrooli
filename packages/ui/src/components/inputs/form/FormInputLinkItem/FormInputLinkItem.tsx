import { LinkItemFormInput, LinkItemFormInputProps, LinkItemType, TranslationFuncCommon, getFormikFieldName } from "@local/shared";
import { Autocomplete, Button, Chip, ListItemIcon, ListItemText, MenuItem, TextField, useTheme } from "@mui/material";
import { LinkInputBase } from "components/inputs/LinkInput/LinkInput";
import { TextInput } from "components/inputs/TextInput/TextInput.js";
import { useField } from "formik";
import { ApiIcon, ArticleIcon, HelpIcon, NoteIcon, ObjectIcon, ProjectIcon, RoutineIcon, SmartContractIcon, TeamIcon, TerminalIcon, UserIcon } from "icons/common.js";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SvgComponent } from "types";
import { CHIP_LIST_LIMIT } from "utils/consts.js";
import { PubSub } from "utils/pubsub.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "../styles.js";
import { FormInputProps } from "../types.js";

type LimitTypeOption = {
    Icon: SvgComponent;
    label: (t: TranslationFuncCommon) => string;
    value: `${LinkItemType}`;
}

const limitTypeOptions: LimitTypeOption[] = [
    {
        Icon: ApiIcon,
        label: t => t("common:Api", { count: 1, defaultValue: "Api" }),
        value: "Api",
    },
    {
        Icon: TerminalIcon,
        label: t => t("common:DataConverter", { count: 1, defaultValue: "Data Converter" }),
        value: "DataConverter",
    },
    {
        Icon: ObjectIcon,
        label: t => t("common:DataStructure", { count: 1, defaultValue: "Data Structure" }),
        value: "DataStructure",
    },
    {
        Icon: NoteIcon,
        label: t => t("common:Note", { count: 1, defaultValue: "Note" }),
        value: "Note",
    },
    {
        Icon: ProjectIcon,
        label: t => t("common:Project", { count: 1, defaultValue: "Project" }),
        value: "Project",
    },
    {
        Icon: ArticleIcon,
        label: t => t("common:Prompt", { count: 1, defaultValue: "Prompt" }),
        value: "Prompt",
    },
    {
        Icon: HelpIcon,
        label: t => t("common:Question", { count: 1, defaultValue: "Question" }),
        value: "Question",
    },
    {
        Icon: RoutineIcon,
        label: t => t("common:RoutineMultiStep", { count: 1, defaultValue: "Multi-step routine (flow)" }),
        value: "RoutineMultiStep",
    },
    {
        Icon: RoutineIcon,
        label: t => t("common:RoutineSingleStep", { count: 1, defaultValue: "Single-step routine (action)" }),
        value: "RoutineSingleStep",
    },
    {
        Icon: SmartContractIcon,
        label: t => t("common:SmartContract", { count: 1, defaultValue: "Smart Contract" }),
        value: "SmartContract",
    },
    {
        Icon: TeamIcon,
        label: t => t("common:Team", { count: 1, defaultValue: "Team" }),
        value: "Team",
    },
    {
        Icon: UserIcon,
        label: t => t("common:User", { count: 1, defaultValue: "User" }),
        value: "User",
    },
];
const acceptedObjectTypes = limitTypeOptions.map(option => option.value);

/** Only accept A-z */
const invalidObjectTypeCharRegex = /[^a-zA-Z]/g;

function withoutInvalidChars(str: string): string {
    return str.replace(invalidObjectTypeCharRegex, "");
}

export function FormInputLinkItem({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<LinkItemFormInput>) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [field, meta, helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));
    const handleChange = useCallback(function handleChangeCallback(value: string) {
        // When editing the config, we're changing the default value
        if (isEditing) {
            const newProps = { ...props, defaultValue: value };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(value);
    }, [isEditing, helpers, props, onConfigUpdate, fieldData]);

    // States for showing additional customization options
    const [showLimits, setShowLimits] = useState(false);
    const [showMore, setShowMore] = useState(false);
    function toggleShowLimits() {
        setShowLimits(showLimits => !showLimits);
        setShowMore(false);
    }
    function toggleShowMore() {
        setShowMore(showMore => !showMore);
        setShowLimits(false);
    }

    const updateProp = useCallback(function updatePropCallback(updatedProps: Partial<LinkItemFormInputProps>) {
        if (isEditing) {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [isEditing, props, onConfigUpdate, fieldData]);

    function updateFieldData(updatedFieldData: Partial<LinkItemFormInput>) {
        if (isEditing) {
            onConfigUpdate({ ...fieldData, ...updatedFieldData });
        }
    }

    const { availableObjectTypes, selectedObjectTypes } = useMemo(() => {
        const available: LimitTypeOption[] = [];
        const selected: LimitTypeOption[] = [];
        limitTypeOptions.forEach(option => {
            if (props.limitTo?.some(limit => limit.type === option.value)) {
                selected.push(option);
            } else {
                available.push(option);
            }
        });
        return { availableObjectTypes: available, selectedObjectTypes: selected };
    }, [props.limitTo]);
    const [limitInputValue, setLimitInputValue] = useState<string>("");
    const handleLimitAdd = useCallback(function handleLimitAddCallback(newType: `${LinkItemType}`) {
        const updatedLimitTo = [...(props.limitTo || [])].filter(limit => limit.type !== newType);
        updatedLimitTo.push({
            type: newType as LinkItemType,
            variant: "any",
        });
        updateProp({ limitTo: updatedLimitTo });
        setLimitInputValue("");
    }, [props.limitTo, updateProp]);
    function handleLimitDelete(index: number) {
        const updatedLimitTo = Array.isArray(props.limitTo) ? [...props.limitTo] : [];
        updatedLimitTo.splice(index, 1);
        updateProp({ limitTo: updatedLimitTo });
    }
    const onObjectTypeInputChange = useCallback(function onObjectTypeInputChangeCallback(change: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) {
        const sanitized = withoutInvalidChars(change.target.value);
        setLimitInputValue(sanitized);
    }, []);
    const onObjectTypeInputKeyDown = useCallback(function onObjectTypeInputKeyDownCallback(event: React.KeyboardEvent<HTMLDivElement>) {
        // Check if the user pressed ',', ';', or enter
        if (!(["Comma", "Enter"].includes(event.code) || (event.code === "Semicolon" && !event.shiftKey))) return;
        event.preventDefault();
        event.stopPropagation();
        // Check if the object type is valid 
        if (!acceptedObjectTypes.includes(limitInputValue as LinkItemType)) {
            PubSub.get().publish("snack", { message: "Invalid object type", severity: "Error" });
            return;
        }
        // Clear input
        setLimitInputValue("");
        // Add to list
        handleLimitAdd(limitInputValue as LinkItemType);
    }, [limitInputValue, handleLimitAdd]);

    const InputElement = useMemo(() => (
        <LinkInputBase
            disabled={disabled}
            fullWidth
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            limitTo={props.limitTo?.map(limit => limit.type) ?? []}
            name={fieldData.fieldName}
            onChange={handleChange}
            placeholder={props.placeholder || (isEditing ? "Enter default value..." : undefined)}
            sxs={{ root: { paddingTop: 1, paddintBottom: 1 } }}
            value={isEditing ? props.defaultValue ?? 0 : field.value}
        />
    ), [disabled, field.value, fieldData.fieldName, handleChange, isEditing, meta.error, meta.touched, props.defaultValue, props.limitTo, props.placeholder]);

    const limitsButtonStyle = useMemo(function limitsButtonStyleMemo() {
        return propButtonWithSectionStyle(showLimits);
    }, [showLimits]);

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
                <Button variant="text" sx={propButtonStyle} onClick={() => updateFieldData({ isRequired: !fieldData.isRequired })}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={limitsButtonStyle} onClick={toggleShowLimits}>
                    Limits
                </Button>
                <Button variant="text" sx={moreButtonStyle} onClick={toggleShowMore}>
                    More
                </Button>
            </FormSettingsButtonRow>
            {showLimits && (
                <FormSettingsSection>
                    <Autocomplete
                        id={`accepted-object-types-input-${fieldData.fieldName}`}
                        fullWidth
                        freeSolo={true}
                        getOptionKey={(option) => typeof option === "string" ? option : option.value}
                        getOptionLabel={(option) => typeof option === "string" ? option : (option.label(t) ?? option.value)}
                        limitTags={CHIP_LIST_LIMIT}
                        multiple
                        options={availableObjectTypes}
                        value={selectedObjectTypes}
                        defaultValue={selectedObjectTypes}
                        renderTags={(value, getTagProps) =>
                            value.map((option, index) => {
                                return (
                                    <Chip
                                        {...getTagProps({ index })}
                                        id={`accepted-object-type-chip-${option.value}`}
                                        key={option.value}
                                        variant="filled"
                                        label={typeof option === "string" ? option : (option.label(t) ?? option.value)}
                                        onDelete={() => handleLimitDelete(index)}
                                        sx={{
                                            backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0",
                                            color: "white",
                                        }}
                                    />
                                );
                            })}
                        renderInput={(params) => (
                            <TextField
                                value={limitInputValue}
                                onChange={onObjectTypeInputChange}
                                label={"Accepted object types"}
                                placeholder={"Routine, Note, Api..."}
                                InputProps={{
                                    ...params.InputProps,
                                    endAdornment: (
                                        <>
                                            {params.InputProps.endAdornment}
                                        </>
                                    ),
                                }}
                                inputProps={params.inputProps}
                                onKeyDown={onObjectTypeInputKeyDown}
                                fullWidth
                                sx={{ paddingRight: 0, minWidth: "250px" }}
                            />
                        )}
                        renderOption={(props, option) => (
                            <MenuItem
                                {...props}
                                onClick={(event) => {
                                    event.preventDefault();
                                    handleLimitAdd(option.value);
                                }}
                                selected={selectedObjectTypes.some(selected => selected.value === option.value)}
                            >
                                <ListItemIcon>
                                    {typeof option === "string" ? null : <option.Icon />}
                                </ListItemIcon>
                                <ListItemText>{option.label(t)}</ListItemText>
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
                        onChange={(event) => { updateProp({ placeholder: event.target.value }); }}
                        value={props.placeholder ?? ""}
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
