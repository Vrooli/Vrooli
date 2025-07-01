import Box from "@mui/material/Box";
import { createTransformFunction, getObjectSlug, getObjectUrlBase, noopSubmit, reportFormConfig, type Report, type ReportFor, type ReportShape } from "@vrooli/shared";
import { Formik, useField } from "formik";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { Dialog } from "../../../components/dialogs/Dialog/Dialog.js";
import { TranslatedAdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { detailsInputFeatures, nameInputFeatures } from "../../../components/inputs/AdvancedInput/styles.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { Selector } from "../../../components/inputs/Selector/Selector.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useStandardUpsertForm } from "../../../hooks/useStandardUpsertForm.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { FormContainer } from "../../../styles.js";
import { type ReportFormProps, type ReportUpsertProps } from "./types.js";

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


function getReasonLabel(reason: string) {
    return ReportReasons[reason] || "";
}

function getDetailsPlaceholder(reason: string) {
    switch (reason) {
        case ReportOptions.Inappropriate:
            return "This content contains inappropriate material such as explicit language, offensive imagery...";
        case ReportOptions.PII:
            return "This content includes personally identifiable information like phone numbers, addresses, email addresses...";
        case ReportOptions.Scam:
            return "This appears to be a scam because it requests personal information, contains suspicious links...";
        case ReportOptions.Spam:
            return "This content appears to be spam because it contains irrelevant advertisements, repetitive messaging...";
        case ReportOptions.Other:
            return "Please explain the issue with this content in detail...";
        default:
            return "Please provide additional details about this report...";
    }
}

function ReportForm({
    disabled,
    display,
    existing,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    values,
    ...props
}: ReportFormProps) {
    const { t } = useTranslation();

    // Use the standardized form hook with config
    const {
        session,
        isLoading,
        handleCancel,
        handleCompleted,
        onSubmit,
        validateValues,
        language,
        languages,
        handleAddLanguage,
        handleDeleteLanguage,
        setLanguage,
        translationErrors,
    } = useStandardUpsertForm({
        objectType: "Report",
        validation: reportFormConfig.validation.schema,
        translationValidation: reportFormConfig.validation.translationSchema,
        transformFunction: createTransformFunction(reportFormConfig),
        endpoints: reportFormConfig.endpoints,
    }, {
        values,
        existing,
        isCreate,
        display,
        disabled,
        isReadLoading,
        isSubmitting: props.isSubmitting,
        handleUpdate: props.handleUpdate,
        setSubmitting: props.setSubmitting,
        onCancel: props.onCancel,
        onCompleted: props.onCompleted,
        onDeleted: props.onDeleted,
        onClose,
    });

    const [reasonField] = useField("reason");

    return display === "Dialog" ? (
        <Dialog
            isOpen={isOpen ?? false}
            onClose={onClose ?? (() => console.warn("onClose not passed to dialog"))}
            size="md"
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Report", { count: 1 })}
                help={t("ReportsHelp")}
            />
            <SearchExistingButton
                href={`/reports${getObjectUrlBase(values.createdFor)}/${getObjectSlug(values.createdFor)}`}
                text={t("ViewExistingReports")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                <FormContainer p={2}>
                    <Box display="flex" flexDirection="column" gap={4}>
                        <Selector
                            name="reason"
                            disabled={isLoading}
                            options={Object.keys(ReportReasons)}
                            getOptionLabel={getReasonLabel}
                            fullWidth
                            label={t("Reason")}
                        />
                        {reasonField.value === ReportOptions.Other && (
                            <TranslatedAdvancedInput
                                features={nameInputFeatures}
                                isRequired={true}
                                language={language}
                                name="otherReason"
                                title={t("ReasonCustom")}
                                placeholder={"Incorrect information, outdated content..."}
                            />
                        )}
                        <TranslatedAdvancedInput
                            features={detailsInputFeatures}
                            isRequired={false}
                            language={language}
                            name="details"
                            title={t("Details")}
                            placeholder={getDetailsPlaceholder(reasonField.value)}
                        />
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                        />
                    </Box>
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={translationErrors}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </Dialog>
    ) : (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Report", { count: 1 })}
                help={t("ReportsHelp")}
            />
            <SearchExistingButton
                href={`/reports${getObjectUrlBase(values.createdFor)}/${getObjectSlug(values.createdFor)}`}
                text={t("ViewExistingReports")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                <FormContainer p={2}>
                    <Box display="flex" flexDirection="column" gap={4}>
                        <Selector
                            name="reason"
                            disabled={isLoading}
                            options={Object.keys(ReportReasons)}
                            getOptionLabel={getReasonLabel}
                            fullWidth
                            label={t("Reason")}
                        />
                        {reasonField.value === ReportOptions.Other && (
                            <TranslatedAdvancedInput
                                features={nameInputFeatures}
                                isRequired={true}
                                language={language}
                                name="otherReason"
                                title={t("ReasonCustom")}
                                placeholder={"Incorrect information, outdated content..."}
                            />
                        )}
                        <TranslatedAdvancedInput
                            features={detailsInputFeatures}
                            isRequired={false}
                            language={language}
                            name="details"
                            title={t("Details")}
                            placeholder={getDetailsPlaceholder(reasonField.value)}
                        />
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                        />
                    </Box>
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={translationErrors}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </>
    );
}

export function ReportUpsert({
    createdFor,
    display,
    isCreate,
    isOpen,
    ...props
}: ReportUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Report, ReportShape>({
        ...reportFormConfig.endpoints.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "Report",
        transform: (existing) => reportFormConfig.transformations.getInitialValues(session.session, existing, createdFor),
    });

    // Ensure we always have valid initial values for new reports
    const formValues = existing || reportFormConfig.transformations.getInitialValues(session.session, {}, createdFor);

    // Simple validation for the wrapper Formik (the actual validation happens in useStandardUpsertForm)
    const validateValues = useCallback(async (values: ReportShape) => {
        // This is just a placeholder - the real validation happens in ReportForm via useStandardUpsertForm
        return {};
    }, []);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={formValues}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <ReportForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={formValues}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
