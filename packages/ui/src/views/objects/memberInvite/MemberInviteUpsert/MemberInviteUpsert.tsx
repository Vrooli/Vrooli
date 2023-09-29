import { DUMMY_ID, endpointGetMemberInvite, endpointPostMemberInvite, endpointPutMemberInvite, MemberInvite, MemberInviteCreateInput, MemberInviteUpdateInput, memberInviteValidation, Session } from "@local/shared";
import { Checkbox, FormControlLabel, TextField } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { MemberInviteFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { MemberInviteShape, shapeMemberInvite } from "utils/shape/models/memberInvite";
import { MemberInviteUpsertProps } from "../types";

/** New resources must include an organization */
export type NewMemberInviteShape = Partial<Omit<MemberInvite, "organization">> & {
    organization: Partial<MemberInvite["organization"]> & ({ id: string })
};

const memberInviteInitialValues = (
    session: Session | undefined,
    existing: NewMemberInviteShape,
): MemberInviteShape => ({
    __typename: "MemberInvite" as const,
    id: DUMMY_ID,
    message: "",
    willBeAdmin: false,
    willHavePermissions: JSON.stringify({}),
    ...existing,
    user: {
        __typename: "User" as const,
        ...existing.user,
        id: existing.user?.id ?? DUMMY_ID,
    },
});

const transformMemberInviteValues = (values: MemberInviteShape, existing: MemberInviteShape, isCreate: boolean) =>
    isCreate ? shapeMemberInvite.create(values) : shapeMemberInvite.update(existing, values);

const validateMemberInviteValues = async (values: MemberInviteShape, existing: MemberInviteShape, isCreate: boolean) => {
    const transformedValues = transformMemberInviteValues(values, existing, isCreate);
    const validationSchema = memberInviteValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const MemberInviteForm = forwardRef<BaseFormRef | undefined, MemberInviteFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

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
                    <RelationshipList
                        isEditing={true}
                        objectType={"MemberInvite"}
                    />
                    <Field
                        fullWidth
                        name="message"
                        label={t("MessageOptional")}
                        as={TextField}
                    />
                    <FormControlLabel
                        control={<Field
                            name="willBeAdmin"
                            type="checkbox"
                            as={Checkbox}
                            size="large"
                            color="secondary"
                        />}
                        label="User will be an administrator"
                    />
                    {/* TODO willHavePermissions */}
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors as any}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});


export const MemberInviteUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: MemberInviteUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const display = toDisplay(isOpen);
    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<MemberInvite, MemberInviteShape>({
        ...endpointGetMemberInvite,
        objectType: "MemberInvite",
        overrideObject: overrideObject as MemberInvite,
        transform: (existing) => memberInviteInitialValues(session, existing as NewMemberInviteShape),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<MemberInvite, MemberInviteCreateInput, MemberInviteUpdateInput>({
        display,
        endpointCreate: endpointPostMemberInvite,
        endpointUpdate: endpointPutMemberInvite,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="member-invite-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateInvite" : "UpdateInvite")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<MemberInviteCreateInput | MemberInviteUpdateInput, MemberInvite>({
                        fetch,
                        inputs: transformMemberInviteValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateMemberInviteValues(values, existing, isCreate)}
            >
                {(formik) => <MemberInviteForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
