import { type LinkUrlFormInput, type LinkUrlFormInputProps, getFormikFieldName } from "@local/shared";
import { Autocomplete, Button, Chip, TextField, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo, useState } from "react";
import { CHIP_LIST_LIMIT } from "../../../utils/consts.js";
import { PubSub } from "../../../utils/pubsub.js";
import { LinkInputBase } from "../LinkInput/LinkInput.js";
import { TextInput } from "../TextInput/TextInput.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "./styles.js";
import { type FormInputProps } from "./types.js";

const MIN_HOST_LENGTH = 2;
const MAX_HOST_LENGTH = 100;
const invalidUrlCharRegex = /[ "<>#%{}|^~[\]`]/g;
const emptyArray = [] as const;

function getLinkKey(url: string): string { return url; }
function getLinkLabel(url: string): string { return url; }

function withoutInvalidChars(url: string): string {
    return url.replace(invalidUrlCharRegex, "");
}


function standardizeUrl(url: string): string {
    try {
        const cleaned = withoutInvalidChars(url);
        const newUrl = new URL(cleaned.includes("://") ? cleaned : `https://${cleaned}`);
        return newUrl.hostname.replace(/^www\./, "") + newUrl.pathname.replace(/\/$/, "");
    } catch (error) {
        return "";
    }
}

export function FormInputLinkUrl({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<LinkUrlFormInput>) {
    const { palette } = useTheme();

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

    const updateProp = useCallback(function updatePropCallback(updatedProps: Partial<LinkUrlFormInputProps>) {
        if (isEditing) {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [isEditing, props, onConfigUpdate, fieldData]);

    function updateFieldData(updatedFieldData: Partial<LinkUrlFormInput>) {
        if (isEditing) {
            onConfigUpdate({ ...fieldData, ...updatedFieldData });
        }
    }

    const [hostInputValue, setHostInputValue] = useState<string>("");
    const handleHostAdd = useCallback(function handleHostAddCallback(newType: string) {
        const standardizedType = standardizeUrl(newType);
        const updatedHosts = Array.isArray(props.acceptedHosts) ? [...props.acceptedHosts].map(host => standardizeUrl(host)) : [];
        if (!updatedHosts.includes(standardizedType)) {
            updatedHosts.push(standardizedType);
            updateProp({ acceptedHosts: updatedHosts });
        }
        setHostInputValue("");
    }, [props.acceptedHosts, updateProp]);
    function handleHostDelete(index: number) {
        const updatedHosts = Array.isArray(props.acceptedHosts) ? [...props.acceptedHosts] : [];
        updatedHosts.splice(index, 1);
        updateProp({ acceptedHosts: updatedHosts });
    }
    const onHostInputChange = useCallback(function onHostInputChangeCallback(change: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) {
        const sanitized = withoutInvalidChars(change.target.value);
        setHostInputValue(sanitized);
    }, []);
    const onHostInputKeyDown = useCallback(function onHostInputKeyDownCallback(event: React.KeyboardEvent<HTMLDivElement>) {
        // Check if the user pressed ',', ';', or enter
        if (!(["Comma", "Enter"].includes(event.code) || (event.code === "Semicolon" && !event.shiftKey))) return;
        event.preventDefault();
        event.stopPropagation();
        // Standardize url
        const url = standardizeUrl(hostInputValue);
        // Check if the url is valid
        if (url.length === 0 && hostInputValue.length > 0) { // Invalid urls are standardize to empty string
            PubSub.get().publish("snack", { message: "Invalid URL", severity: "Error" });
            return;
        }
        // Check if url is valid length
        if (url.length < MIN_HOST_LENGTH) {
            PubSub.get().publish("snack", { message: "Url too short", severity: "Error" });
            return;
        }
        if (url.length > MAX_HOST_LENGTH) {
            PubSub.get().publish("snack", { message: "Url too long", severity: "Error" });
            return;
        }
        // Clear input
        setHostInputValue("");
        // Check if url is already selected
        if ((props.acceptedHosts ?? []).map(host => standardizeUrl(host)).includes(url)) {
            PubSub.get().publish("snack", { message: "Url already selected", severity: "Error" });
            return;
        }
        // Add url to list
        handleHostAdd(url);
    }, [hostInputValue, handleHostAdd, props.acceptedHosts]);

    const InputElement = useMemo(() => (
        <LinkInputBase
            disabled={disabled}
            fullWidth
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            name={fieldData.fieldName}
            onChange={handleChange}
            placeholder={props.placeholder || (isEditing ? "Enter default value..." : undefined)}
            sxs={{ root: { paddingTop: 1, paddintBottom: 1 } }}
            value={isEditing ? props.defaultValue ?? 0 : field.value}
        />
    ), [disabled, field.value, fieldData.fieldName, handleChange, isEditing, meta.error, meta.touched, props.defaultValue, props.placeholder]);

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
                        id={`accepted-hosts-input-${fieldData.fieldName}`}
                        fullWidth
                        freeSolo={true}
                        getOptionKey={getLinkKey}
                        getOptionLabel={getLinkLabel}
                        limitTags={CHIP_LIST_LIMIT}
                        multiple
                        options={emptyArray}
                        value={props.acceptedHosts || []}
                        defaultValue={props.acceptedHosts || []}
                        renderTags={(value, getTagProps) =>
                            value.map((option, index) => {
                                return (
                                    <Chip
                                        {...getTagProps({ index })}
                                        id={`accepted-hosts-chip-${option}`}
                                        key={option}
                                        variant="filled"
                                        label={option}
                                        onDelete={() => handleHostDelete(index)}
                                        sx={{
                                            backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0",
                                            color: "white",
                                        }}
                                    />
                                );
                            })}
                        renderInput={(params) => (
                            <TextField
                                value={hostInputValue}
                                onChange={onHostInputChange}
                                label={"Accepted hosts"}
                                placeholder={"example.com"}
                                InputProps={{
                                    ...params.InputProps,
                                    endAdornment: (
                                        <>
                                            {params.InputProps.endAdornment}
                                        </>
                                    ),
                                }}
                                inputProps={params.inputProps}
                                onKeyDown={onHostInputKeyDown}
                                fullWidth
                                sx={{ paddingRight: 0, minWidth: "250px" }}
                            />
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
