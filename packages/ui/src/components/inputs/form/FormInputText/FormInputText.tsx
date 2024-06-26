import { Button, Slider, Typography } from "@mui/material";
import { IntegerInputBase } from "components/inputs/IntegerInput/IntegerInput";
import { RichInputBase } from "components/inputs/RichInput/RichInput";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { useField } from "formik";
import { TextFormInput, TextFormInputProps } from "forms/types";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormInputProps } from "../types";

/**
 * Options for the autoComplete attribute in text inputs. 
 * NOTE: We purposely omit password and credit-card related options, as they could be a security risk.
 */
const autoCompleteOptions = [
    "off", "on", "name", "honorific-prefix", "given-name",
    "additional-name", "family-name", "honorific-suffix",
    "nickname", "email", "username", "one-time-code",
    "organization-title", "organization", "street-address",
    "address-line1", "address-line2", "address-line3",
    "address-level4", "address-level3", "address-level2",
    "address-level1", "country", "country-name", "postal-code",
    "transaction-currency", "transaction-amount", "language",
    "bday", "bday-day", "bday-month", "bday-year", "sex", "url", "photo",
]; // Omitted: current-password, new-password, cc-name, cc-given-name, cc-additional-name, cc-family-name, cc-number, cc-exp, cc-exp-month, cc-exp-year, cc-csc, cc-type

const propButtonStyle = { textDecoration: "underline", textTransform: "none" } as const;

export function FormInputText({
    disabled,
    fieldData,
    onConfigUpdate,
}: FormInputProps<TextFormInput>) {
    const { t } = useTranslation();

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

    function updateProp(updatedProps: Partial<TextFormInputProps>) {
        if (typeof onConfigUpdate !== "function") {
            return;
        }
        const newProps = { ...props, ...updatedProps };
        onConfigUpdate({ ...fieldData, props: newProps });
    }
    function updateFieldData(updatedFieldData: Partial<TextFormInput>) {
        if (typeof onConfigUpdate !== "function") {
            return;
        }
        onConfigUpdate({ ...fieldData, ...updatedFieldData });
    }

    // Determine if rich input or normal text input will be used
    const InputElement = useMemo(function inputElementMemo() {
        const commonProps = {
            disabled,
            error: meta.touched && !!meta.error,
            helperText: meta.touched && meta.error,
            name: fieldData.fieldName,
            onBlur: field.onBlur,
            placeholder: props.placeholder || (typeof onConfigUpdate === "function" ? "Enter default value..." : undefined),
            value: typeof onConfigUpdate === "function" ? props.defaultValue : field.value,
        } as const;
        if (props.isMarkdown) {
            return (
                <RichInputBase
                    disableAssistant={typeof onConfigUpdate === "function"}
                    maxChars={props.maxChars}
                    maxRows={props.maxRows}
                    minRows={props.minRows}
                    onChange={handleChange}
                    {...commonProps}
                />
            );
        }
        const multiLineProps = props.maxRows ? { multiline: true, rows: props.maxRows } : {};
        return <TextInput
            fullWidth
            onChange={(event) => { handleChange(event.target.value); }}
            {...multiLineProps}
            {...commonProps}
        />;
    }, [disabled, meta.touched, meta.error, fieldData.fieldName, field.onBlur, field.value, onConfigUpdate, props.placeholder, props.defaultValue, props.isMarkdown, props.maxRows, props.maxChars, props.minRows, handleChange]);

    // If we're not editing, just display the input element
    if (typeof onConfigUpdate !== "function") {
        return InputElement;
    }
    // If we're editing, display the input element with buttons to change props
    return (
        <div>
            {InputElement}
            <div style={{ display: "flex", flexDirection: "row" }}>
                <Button variant="text" sx={propButtonStyle} onClick={() => { updateFieldData({ isRequired: !fieldData.isRequired }); }}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={() => { updateProp({ isMarkdown: !props.isMarkdown }); }}>
                    {props.isMarkdown ? "Text" : "Markdown"}
                </Button>
                <Button variant="text" sx={{ ...propButtonStyle, color: showLimits ? "primary.main" : undefined }} onClick={toggleShowLimits}>
                    Limits
                </Button>
                <Button variant="text" sx={{ ...propButtonStyle, color: showMore ? "primary.main" : undefined }} onClick={toggleShowMore}>
                    More
                </Button>
            </div>
            {/* Set max chars, max rows, etc. */}
            {showLimits && (
                <div style={{ marginTop: "10px", padding: "8px", display: "flex", flexDirection: "column", gap: "8px" }}>
                    <IntegerInputBase
                        fullWidth
                        label={"Max text length"}
                        name="maxChars"
                        onChange={(value) => { updateProp({ maxChars: value }); }}
                        tooltip="The maximum number of characters allowed in the text field"
                        value={props.maxChars ?? 0}
                    />
                    <div>
                        <Typography id={`${fieldData.fieldName}-rows-slider`} gutterBottom>
                            {t("MinRows")}/{t("MaxRows")}
                        </Typography>
                        <Slider
                            aria-labelledby={`${fieldData.fieldName}-rows-slider`}
                            onChange={(_, value) => { updateProp({ minRows: value[0], maxRows: value[1] }); }}
                            min={1}
                            max={20}
                            marks={[
                                {
                                    value: 1,
                                    label: "1",
                                },
                                {
                                    value: 20,
                                    label: "20",
                                },
                            ]}
                            sx={{
                                marginLeft: 1.5,
                                marginRight: 1.5,
                                width: "-webkit-fill-available",
                            }}
                            value={[props.minRows ?? 1, props.maxRows ?? 1]}
                            valueLabelDisplay="auto"
                        />
                    </div>
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
                    <SelectorBase
                        fullWidth
                        getOptionLabel={(option) => option}
                        inputAriaLabel="autoComplete"
                        label="Auto-fill"
                        name="autoComplete"
                        onChange={(value) => { updateProp({ autoComplete: value }); }}
                        options={autoCompleteOptions}
                        value={props.autoComplete ?? "off"}
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
