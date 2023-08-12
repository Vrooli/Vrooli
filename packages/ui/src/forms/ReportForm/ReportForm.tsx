import { DUMMY_ID, Report, ReportFor, reportValidation, Session } from "@local/shared";
import { TextField } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { Selector } from "components/inputs/Selector/Selector";
import { Field, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReportFormProps } from "forms/types";
import { forwardRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ReportShape, shapeReport } from "utils/shape/models/report";

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

/** New resources must include a createdFor __typename and ID */
export type NewReportShape = Partial<Omit<Report, "createdFor">> & { createdFor: { __typename: ReportFor, id: string } };

export const reportInitialValues = (
    session: Session | undefined,
    existing: NewReportShape,
): ReportShape => ({
    __typename: "Report" as const,
    id: DUMMY_ID,
    reason: "",
    otherReason: "",
    details: "",
    language: getUserLanguages(session)[0],
    ...existing,
});

export const transformReportValues = (values: ReportShape, existing: ReportShape, isCreate: boolean) =>
    isCreate ? shapeReport.create(values) : shapeReport.update(existing, values);

export const validateReportValues = async (values: ReportShape, existing: ReportShape, isCreate: boolean) => {
    const transformedValues = transformReportValues(values, existing, isCreate);
    console.log("transformed report", transformedValues);
    const validationSchema = reportValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const ReportForm = forwardRef<BaseFormRef | undefined, ReportFormProps>(({
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
                display={display}
                isLoading={isLoading}
                maxWidth={700}
                ref={ref}
            >
                <FormContainer>
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
                        label={t("Reason")}
                    />
                    {reasonField.value === ReportOptions.Other && <Field
                        fullWidth
                        name="otherReason"
                        label={t("ReasonCustom")}
                        helperText={t("ReasonCustomHelp")}
                        as={TextField}
                    />}
                    <MarkdownInput
                        maxChars={8192}
                        maxRows={10}
                        minRows={4}
                        name="details"
                        placeholder={t("DetailsOptional")}
                        zIndex={zIndex}
                    />
                </FormContainer>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={props.errors as any}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
                zIndex={zIndex}
            />
        </>
    );
});
