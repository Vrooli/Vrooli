import { DUMMY_ID, endpointGetReport, endpointPostReport, endpointPutReport, noopSubmit, Report, ReportCreateInput, ReportFor, ReportUpdateInput, reportValidation, Session } from "@local/shared";
import { Link, TextField, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RichInput } from "components/inputs/RichInput/RichInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { formNavLink } from "forms/styles";
import { ReportFormProps } from "forms/types";
import { useConfirmBeforeLeave } from "hooks/useConfirmBeforeLeave";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { clickSize, FormContainer } from "styles";
import { getYou } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ReportShape, shapeReport } from "utils/shape/models/report";
import { ReportUpsertProps } from "../types";

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
    const validationSchema = reportValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const ReportForm = ({
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onCompleted,
    onDeleted,
    values,
    ...props
}: ReportFormProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const session = useContext(SessionContext);

    /**
     * Opens existing reports in a new tab
     */
    const toExistingReports = useCallback(() => {
        window.open("/reports", "_blank");// TODO change url
    }, []);

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: [],
    });

    const [reasonField] = useField("reason");

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Report, ReportCreateInput, ReportUpdateInput>({
        display,
        endpointCreate: endpointPostReport,
        endpointUpdate: endpointPutReport,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { handleClose } = useConfirmBeforeLeave({ handleCancel, shouldPrompt: dirty });
    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && !existing) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ReportCreateInput | ReportUpdateInput, Report>({
            fetch,
            inputs: transformReportValues(values, existing, isCreate),
            onSuccess: (data) => { handleCompleted(data); },
            onCompleted: () => { props.setSubmitting(false); },
        });
    }, [disabled, existing, fetch, handleCompleted, isCreate, props, values]);

    return (
        <MaybeLargeDialog
            display={display}
            id="report-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t("Report", { count: 1 })}
                help={t("ReportsHelp")}
            />
            <Link onClick={toExistingReports}>
                <Typography sx={{
                    ...clickSize,
                    ...formNavLink,
                    justifyContent: "center",
                    marginTop: 2,
                }}>
                    {t("ViewExistingReports")}
                </Typography>
            </Link>
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer sx={{ gap: 2 }}>
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
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
                    <RichInput
                        maxChars={8192}
                        maxRows={10}
                        minRows={4}
                        name="details"
                        placeholder={t("DetailsOptional")}
                    />
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors as any}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const ReportUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ReportUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<Report, ReportShape>({
        ...endpointGetReport,
        objectType: "Report",
        overrideObject,
        transform: (existing) => reportInitialValues(session, existing as NewReportShape),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateReportValues(values, existing, isCreate)}
        >
            {(formik) => <ReportForm
                disabled={!(isCreate || canUpdate)}
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
};
