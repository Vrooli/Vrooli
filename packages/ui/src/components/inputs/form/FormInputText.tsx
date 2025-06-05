import { Button, Slider, Typography } from "@mui/material";
import { getFormikFieldName, type TextFormInput, type TextFormInputProps } from "@vrooli/shared";
import { useField } from "formik";
import { useCallback, useMemo, useState, type ChangeEvent } from "react";
import { useTranslation } from "react-i18next";
import { IntegerInputBase } from "../IntegerInput/IntegerInput.js";
import { RichInputBase } from "../RichInput/RichInput.js";
import { SelectorBase } from "../Selector/Selector.js";
import { TextInput } from "../TextInput/TextInput.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "./styles.js";
import { type FormInputProps } from "./types.js";

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

function getAutocompleteOptionLabel(option: string) {
    return option;
}

const sliderMarks = [
    {
        value: 1,
        label: "1",
    },
    {
        value: 20,
        label: "20",
    },
];

const sliderStyle = {
    marginLeft: 1.5,
    marginRight: 1.5,
    width: "-webkit-fill-available",
} as const;

export function FormInputText({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<TextFormInput>) {
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

    function updateProp(updatedProps: Partial<TextFormInputProps>) {
        if (!isEditing) {
            return;
        }
        const newProps = { ...props, ...updatedProps };
        onConfigUpdate({ ...fieldData, props: newProps });
    }
    function updateFieldData(updatedFieldData: Partial<TextFormInput>) {
        if (!isEditing) {
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
            placeholder: props.placeholder || (isEditing ? "Enter default value..." : undefined),
            value: isEditing ? props.defaultValue : field.value,
        } as const;

        function onChange(event: ChangeEvent<HTMLInputElement>) {
            handleChange(event.target.value);
        }

        if (props.isMarkdown) {
            return (
                <RichInputBase
                    disableAssistant={isEditing}
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
            onChange={onChange}
            {...multiLineProps}
            {...commonProps}
        />;
    }, [disabled, isEditing, meta.touched, meta.error, fieldData.fieldName, field.onBlur, field.value, props.placeholder, props.defaultValue, props.isMarkdown, props.maxRows, props.maxChars, props.minRows, handleChange]);

    const limitsButtonStyle = useMemo(function limitsButtonStyleMemo() {
        return propButtonWithSectionStyle(showLimits);
    }, [showLimits]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    // If we're not editing, just display the input element
    if (!isEditing) {
        return InputElement;
    }
    // If we're editing, display the input element with buttons to change props
    return (
        <div>
            {InputElement}
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={() => { updateFieldData({ isRequired: !fieldData.isRequired }); }}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={() => { updateProp({ isMarkdown: !props.isMarkdown }); }}>
                    {props.isMarkdown ? "Text" : "Markdown"}
                </Button>
                <Button variant="text" sx={limitsButtonStyle} onClick={toggleShowLimits}>
                    Limits
                </Button>
                <Button variant="text" sx={moreButtonStyle} onClick={toggleShowMore}>
                    More
                </Button>
            </FormSettingsButtonRow>
            {/* Set max chars, max rows, etc. */}
            {showLimits && (
                <FormSettingsSection>
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
                            marks={sliderMarks}
                            sx={sliderStyle}
                            value={[props.minRows ?? 1, props.maxRows ?? 1]}
                            valueLabelDisplay="auto"
                        />
                    </div>
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
                    <SelectorBase
                        fullWidth
                        getOptionLabel={getAutocompleteOptionLabel}
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
                </FormSettingsSection>
            )}
        </div>
    );
}
