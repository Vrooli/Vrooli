import { DUMMY_ID, endpointsReport, getObjectSlug, getObjectUrlBase, noopSubmit, type Report, type ReportCreateInput, type ReportFor, type ReportShape, type ReportUpdateInput, reportValidation, type Session, shapeReport } from "@local/shared";
import { Box } from "@mui/material";
import { Formik, useField } from "formik";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { TranslatedAdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { detailsInputFeatures, nameInputFeatures } from "../../../components/inputs/AdvancedInput/styles.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { Selector } from "../../../components/inputs/Selector/Selector.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { FormContainer } from "../../../styles.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
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

export function reportInitialValues(
    session: Session | undefined,
    existing: Omit<Partial<ReportShape>, "createdFor">,
    createdFor: { __typename: ReportFor, id: string },
): ReportShape {
    return {
        ...existing,
        __typename: "Report" as const,
        id: DUMMY_ID,
        reason: "",
        otherReason: "",
        details: "",
        language: getUserLanguages(session)[0],
        createdFor,
    };
}

export function transformReportValues(values: ReportShape, existing: ReportShape, isCreate: boolean) {
    return isCreate ? shapeReport.create(values) : shapeReport.update(existing, values);
}

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
    const session = useContext(SessionContext);

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
    });

    const [reasonField] = useField("reason");

    const { handleCancel, handleCompleted } = useUpsertActions<Report>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Report",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Report, ReportCreateInput, ReportUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsReport.createOne,
        endpointUpdate: endpointsReport.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Report" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<ReportCreateInput | ReportUpdateInput, Report>({
        disabled,
        existing,
        fetch,
        inputs: transformReportValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="report-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
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
                errors={props.errors}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
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
        ...endpointsReport.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "Report",
        transform: (existing) => reportInitialValues(session, existing, createdFor),
    });

    async function validateValues(values: ReportShape) {
        if (!existing) return;
        return await validateFormValues(values, { ...existing, createdFor }, isCreate, transformReportValues, reportValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <ReportForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={existing}
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
