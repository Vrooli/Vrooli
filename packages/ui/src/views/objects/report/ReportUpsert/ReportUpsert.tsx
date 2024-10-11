import { DUMMY_ID, endpointGetReport, endpointPostReport, endpointPutReport, getObjectSlug, getObjectUrlBase, noopSubmit, Report, ReportCreateInput, ReportFor, ReportShape, ReportUpdateInput, reportValidation, Session, shapeReport } from "@local/shared";
import { useSubmitHelper } from "api/fetchWrapper";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RichInput } from "components/inputs/RichInput/RichInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useManagedObject } from "hooks/useManagedObject";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getUserLanguages } from "utils/display/translationTools";
import { validateFormValues } from "utils/validateFormValues";
import { ReportFormProps, ReportUpsertProps } from "../types";

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
        endpointCreate: endpointPostReport,
        endpointUpdate: endpointPutReport,
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
                <FormContainer>
                    <FormSection variant="transparent">
                        <Selector
                            name="reason"
                            disabled={isLoading}
                            options={Object.keys(ReportReasons)}
                            getOptionLabel={getReasonLabel}
                            fullWidth
                            label={t("Reason")}
                        />
                        {reasonField.value === ReportOptions.Other && <Field
                            fullWidth
                            name="otherReason"
                            label={t("ReasonCustom")}
                            helperText={t("ReasonCustomHelp")}
                            as={TextInput}
                        />}
                        <RichInput
                            isRequired={false}
                            maxChars={8192}
                            maxRows={10}
                            minRows={4}
                            name="details"
                            placeholder={t("Details")}
                        />
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                        />
                    </FormSection>
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
        ...endpointGetReport,
        disabled: display === "dialog" && isOpen !== true,
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
