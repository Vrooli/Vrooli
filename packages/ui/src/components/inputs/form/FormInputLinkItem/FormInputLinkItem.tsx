import { GqlModelType } from "@local/shared";
import { Autocomplete, Button, Chip, TextField, useTheme } from "@mui/material";
import { LinkInputBase } from "components/inputs/LinkInput/LinkInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { useField } from "formik";
import { LinkItemFormInput, LinkItemFormInputProps, LinkItemLimitTo } from "forms/types";
import { TFunction } from "i18next";
import { useCallback, useMemo, useState } from "react";
import { PubSub } from "utils/pubsub";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { FormInputProps } from "../types";

// All = "All",
// Api = "Api",
// Code = "Code",
// Note = "Note",
// Project = "Project",
// Question = "Question",
// Routine = "Routine",
// Standard = "Standard",
// Team = "Team",
// User = "User",

type LimitTypeOption = {
    label: (t: TFunction<"common", undefined, "common">) => string;
    value: LinkItemLimitTo["type"];
}

const limitTypeOptions: LimitTypeOption[] = [{
    label: t => t("common:Api", { count: 1, defaultValue: "Api" }),
    value: SearchPageTabOption.Api,
}, {
    label: t => t("common:Code", { count: 1, defaultValue: "Code" }),
    value: SearchPageTabOption.Code,
}, {
    label: t => t("common:Note", { count: 1, defaultValue: "Note" }),
    value: SearchPageTabOption.Note,
}, {
    label: t => t("common:Project", { count: 1, defaultValue: "Project" }),
    value: SearchPageTabOption.Project,
}, {
    label: t => t("common:Question", { count: 1, defaultValue: "Question" }),
    value: SearchPageTabOption.Question,
}, {
    label: t => t("common:Routine", { count: 1, defaultValue: "Routine" }),
    value: SearchPageTabOption.Routine,
}, {
    label: t => t("common:Standard", { count: 1, defaultValue: "Standard" }),
    value: SearchPageTabOption.Standard,
}, {
    label: t => t("common:Team", { count: 1, defaultValue: "Team" }),
    value: SearchPageTabOption.Team,
}, {
    label: t => t("common:User", { count: 1, defaultValue: "User" }),
    value: SearchPageTabOption.User,
}];

const propButtonStyle = { textDecoration: "underline", textTransform: "none" } as const;

const MIN_HOST_LENGTH = 2;
const MAX_HOST_LENGTH = 100;
const invalidUrlCharRegex = /[ "<>#%{}|^~[\]`]/g;

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

export function FormInputLinkItem({
    disabled,
    fieldData,
    onConfigUpdate,
}: FormInputProps<LinkItemFormInput>) {
    const { palette, typography } = useTheme();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [field, meta, helpers] = useField(fieldData.fieldName);
    const handleChange = useCallback(function handleChangeCallback(value: string) {
        // When editing the config, we're changing the default value
        if (typeof onConfigUpdate === "function") {
            const newProps = { ...props, defaultValue: value };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(value);
    }, [onConfigUpdate, props, fieldData, helpers]);

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
        if (typeof onConfigUpdate === "function") {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [onConfigUpdate, props, fieldData]);

    function updateFieldData(updatedFieldData: Partial<LinkItemFormInput>) {
        if (typeof onConfigUpdate === "function") {
            onConfigUpdate({ ...fieldData, ...updatedFieldData });
        }
    }

    const [limitInputValue, setLimitInputValue] = useState<string>("");
    const handleLimitAdd = useCallback(function handleLimitAddCallback(newType: string) {
        const standardizedType = standardizeUrl(newType);
        const updatedLimitTo = Array.isArray(props.limitTo) ? [...props.limitTo].map(host => standardizeUrl(host)) : [];
        if (!updatedLimitTo.includes(standardizedType)) {
            updatedLimitTo.push(standardizedType);
            updateProp({ limitTo: updatedLimitTo as GqlModelType[] });
        }
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
        //TODO
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
        setLimitInputValue("");
        // Check if url is already selected
        if ((props.limitTo ?? []).map(host => standardizeUrl(host)).includes(url)) {
            PubSub.get().publish("snack", { message: "Url already selected", severity: "Error" });
            return;
        }
        // Add url to list
        handleLimitAdd(url);
    }, [limitInputValue, handleLimitAdd, props.limitTo]);

    const InputElement = useMemo(() => (
        <LinkInputBase
            disabled={disabled}
            fullWidth
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            name={fieldData.fieldName}
            onChange={handleChange}
            placeholder={props.placeholder || (typeof onConfigUpdate === "function" ? "Enter default value..." : undefined)}
            sxs={{ root: { paddingTop: 1, paddintBottom: 1 } }}
            value={typeof onConfigUpdate === "function" ? props.defaultValue ?? 0 : field.value}
        />
    ), [disabled, field.value, fieldData.fieldName, handleChange, meta.error, meta.touched, onConfigUpdate, props.defaultValue, props.placeholder]);

    if (typeof onConfigUpdate !== "function") {
        return InputElement;
    }

    return (
        <div>
            {InputElement}
            <div style={{ display: "flex", flexDirection: "row" }}>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateFieldData({ isRequired: !fieldData.isRequired })}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={{ ...propButtonStyle, color: showLimits ? "primary.main" : undefined }} onClick={toggleShowLimits}>
                    Limits
                </Button>
                <Button variant="text" sx={{ ...propButtonStyle, color: showMore ? "primary.main" : undefined }} onClick={toggleShowMore}>
                    More
                </Button>
            </div>
            {showLimits && (
                <div style={{ marginTop: "10px", padding: "8px", display: "flex", flexDirection: "column", gap: "8px" }}>
                    <Autocomplete
                        id={`accepted-object-types-input-${fieldData.fieldName}`}
                        fullWidth
                        freeSolo={true}
                        getOptionKey={(option) => option}
                        getOptionLabel={(option) => option}
                        limitTags={3}
                        multiple
                        options={[]}
                        value={props.limitTo || []}
                        defaultValue={props.limitTo || []}
                        renderTags={(value, getTagProps) =>
                            value.map((option, index) => {
                                return (
                                    <Chip
                                        {...getTagProps({ index })}
                                        id={`tag-chip-${option}`}
                                        key={option}
                                        variant="filled"
                                        label={option}
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
                    />
                </div>
            )}
            {showMore && (
                <div style={{ marginTop: "10px", padding: "8px", display: "flex", flexDirection: "column", gap: "8px" }}>
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
                </div>
            )}
        </div>
    );
}
