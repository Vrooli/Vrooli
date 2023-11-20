import { DUMMY_ID, endpointGetReport, endpointPostReport, endpointPutReport, noopSubmit, Report, ReportCreateInput, ReportFor, ReportUpdateInput, reportValidation, Session } from "@local/shared";
import { Link, Typography } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RichInput } from "components/inputs/RichInput/RichInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { formNavLink } from "forms/styles";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { clickSize, FormContainer } from "styles";
import { getYou } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { ReportShape, shapeReport } from "utils/shape/models/report";
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

const ReportForm = ({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: ReportFormProps) => {
    const { t } = useTranslation();
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

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<Report>({
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
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "Report" });

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
                style={{ paddingBottom: "64px" }}
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
                        as={TextInput}
                    />}
                    <RichInput
                        isOptional
                        maxChars={8192}
                        maxRows={10}
                        minRows={4}
                        name="details"
                        placeholder={t("Details")}
                    />
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
        isCreate,
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
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformReportValues, reportValidation)}
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
