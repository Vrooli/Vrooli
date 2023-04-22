import { TextField } from "@mui/material";
import { Report, ReportFor, Session } from "@shared/consts";
import { DUMMY_ID } from "@shared/uuid";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { SelectLanguageMenu } from "../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { Selector } from "../../components/inputs/Selector/Selector";
import { Field, useField } from "formik";
import { BaseForm } from "../../forms/BaseForm/BaseForm";
import { ReportFormProps } from "../../forms/types";
import { forwardRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "../../utils/display/translationTools";
import { ReportShape } from "../../utils/shape/models/report";

enum ReportOptions {
    Inappropriate = "Inappropriate",
    PII = "PII",
    Scam = "Scam",
    Spam = "Spam",
    Other = "Other",
}

const ReportReasons = {
    [ReportOptions.Inappropriate]: "Inappropriate Content",
    [ReportOptions.PII]: "Includes Personally Identifiable Information (PII)",
    [ReportOptions.Scam]: "Scam",
    [ReportOptions.Spam]: "Spam",
    [ReportOptions.Other]: "Other",
};

export const reportInitialValues = (
    session: Session | undefined,
    createdFor: ReportFor,
    createdForId: string,
    existing?: Report | null | undefined,
): ReportShape => ({
    __typename: "Report" as const,
    id: DUMMY_ID,
    createdFor: { __typename: createdFor, id: createdForId },
    reason: "",
    otherReason: "",
    details: "",
    language: getUserLanguages(session)[0],
    ...existing,
});


export const ReportForm = forwardRef<any, ReportFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
    const { t } = useTranslation();

    const [languageField, , languageHelpers] = useField("language");
    const setLanguage = useCallback((language: string) => {
        languageHelpers.setValue(language);
    }, [languageHelpers]);

    const [reasonField] = useField("reason");

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(600px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <SelectLanguageMenu
                    currentLanguage={languageField.value}
                    handleCurrent={setLanguage}
                    languages={[languageField.value]}
                    zIndex={zIndex}
                />
                <Selector
                    name="reason"
                    disabled={isLoading}
                    options={Object.keys(ReportReasons)}
                    getOptionLabel={(r) => ReportReasons[r]}
                    fullWidth
                    inputAriaLabel="select reason"
                    label={t("Reason")}
                />
                {reasonField.value === ReportOptions.Other && <Field
                    fullWidth
                    name="otherReason"
                    label={t("ReasonCustom")}
                    helperText={t("ReasonCustomHelp")}
                    as={TextField}
                />}
                <Field
                    fullWidth
                    multiline
                    rows={4}
                    name="details"
                    label={t("DetailsOptional")}
                    helperText={t("ReportDetailsHelp")}
                    as={TextField}
                />
                <GridSubmitButtons
                    display={display}
                    errors={props.errors as any}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    );
});
