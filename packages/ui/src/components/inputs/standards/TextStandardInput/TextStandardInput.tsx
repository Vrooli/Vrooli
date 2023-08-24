import { Grid, Slider, TextField, Typography } from "@mui/material";
import { CheckboxInput } from "components/inputs/CheckboxInput/CheckboxInput";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { Selector } from "components/inputs/Selector/Selector";
import { Field, useFormikContext } from "formik";
import { FieldDataText } from "forms/types";
import { useTranslation } from "react-i18next";
import { TextStandardInputProps } from "../types";

const autoCompleteOptions = [
    "off", "on", "name", "honorific-prefix", "given-name",
    "additional-name", "family-name", "honorific-suffix",
    "nickname", "email", "username", "new-password",
    "current-password", "one-time-code", "organization-title",
    "organization", "street-address", "address-line1",
    "address-line2", "address-line3", "address-level4",
    "address-level3", "address-level2", "address-level1",
    "country", "country-name", "postal-code", "cc-name",
    "cc-given-name", "cc-additional-name", "cc-family-name",
    "cc-number", "cc-exp", "cc-exp-month", "cc-exp-year",
    "cc-csc", "cc-type", "transaction-currency",
    "transaction-amount", "language", "bday", "bday-day",
    "bday-month", "bday-year", "sex", "url", "photo",
];

/**
 * Input for entering (and viewing format of) TextField data that 
 * must match a certain schema.
 */
export const TextStandardInput = ({
    isEditing,
}: TextStandardInputProps) => {
    const { t } = useTranslation();

    const { values, setFieldValue } = useFormikContext<FieldDataText["props"]>();

    const handleSliderChange = (_event, newValue) => {
        setFieldValue("minRows", newValue[0]);
        setFieldValue("maxRows", newValue[1]);
    };

    return (
        <Grid container spacing={2}>
            <Grid item xs={12}>
                <Field
                    name="isMarkdown"
                    label={"Support markdown?"}
                    component={CheckboxInput}
                />
            </Grid>
            <Grid item xs={12}>
                {values.isMarkdown ?
                    <MarkdownInput
                        disabled={!isEditing}
                        name="defaultValue"
                        placeholder={t("DefaultValue")}
                        minRows={values.minRows ?? 2}
                        maxRows={values.maxRows ?? 4}
                    /> :
                    <Field
                        fullWidth
                        disabled={!isEditing}
                        name="defaultValue"
                        label={t("DefaultValue")}
                        as={TextField}
                    />
                }
            </Grid>
            <Grid item xs={12}>
                <Selector
                    name="autoComplete"
                    disabled={!isEditing}
                    options={autoCompleteOptions}
                    getOptionLabel={(option) => option}
                    fullWidth
                    inputAriaLabel="autoComplete"
                    label="Auto-fill"
                />
            </Grid>
            <Grid item xs={12} md={6}>
                <IntegerInput
                    disabled={!isEditing}
                    fullWidth
                    label={"Max text length"}
                    name="maxChars"
                    tooltip="The maximum number of characters allowed in the text field"
                />
            </Grid>
            <Grid item xs={12} md={6} sx={{ margin: "auto" }}>
                <Typography id="range-slider" gutterBottom>
                    {t("MinRows")}/{t("MaxRows")}
                </Typography>
                <Slider
                    aria-labelledby="range-slider"
                    disabled={!isEditing}
                    value={[values.minRows ?? 1, values.maxRows ?? 1]}
                    onChange={handleSliderChange}
                    valueLabelDisplay="auto"
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
                    }}
                />
            </Grid>

        </Grid>
    );
};
