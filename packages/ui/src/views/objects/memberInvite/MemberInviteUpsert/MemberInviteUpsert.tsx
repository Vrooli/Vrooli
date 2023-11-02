import { DUMMY_ID, endpointPostMemberInvites, endpointPutMemberInvites, MemberInvite, MemberInviteCreateInput, MemberInviteUpdateInput, memberInviteValidation, noop, noopSubmit, Session } from "@local/shared";
import { Box, Checkbox, FormControlLabel, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Field, Formik } from "formik";
import { MemberInviteFormProps } from "forms/types";
import { useConfirmBeforeLeave } from "hooks/useConfirmBeforeLeave";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
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

const transformMemberInviteValues = (values: MemberInviteShape[], existing: MemberInviteShape[], isCreate: boolean) =>
    isCreate ?
        values.map((value) => shapeMemberInvite.create(value)) :
        values.map((value, index) => shapeMemberInvite.update(existing[index], value)); // Assumes the dialog doesn't change the order or remove items

const validateMemberInviteValues = async (values: MemberInviteShape[], existing: MemberInviteShape[], isCreate: boolean) => {
    const transformedValues = transformMemberInviteValues(values, existing, isCreate);
    const validationSchema = memberInviteValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await Promise.all(transformedValues.map(async (value) => await validateAndGetYupErrors(validationSchema, value)));

    // Filter and combine the result into one object with only error results
    const combinedResult = result.reduce((acc, curr, index) => {
        if (Object.keys(curr).length > 0) {  // check if the object has any keys (errors)
            acc[index] = curr;
        }
        return acc;
    }, {} as any);

    return combinedResult;
};

const MemberInviteForm = ({
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
}: MemberInviteFormProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const { palette } = useTheme();
    const [message, setMessage] = useState("");

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<MemberInvite[], MemberInviteCreateInput[], MemberInviteUpdateInput[]>({
        display,
        endpointCreate: endpointPostMemberInvites,
        endpointUpdate: endpointPutMemberInvites,
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
        if (!isCreate && existing.length === 0) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        fetchLazyWrapper<MemberInviteCreateInput[] | MemberInviteUpdateInput[], MemberInvite[]>({
            fetch,
            inputs: transformMemberInviteValues(values, existing, isCreate) as MemberInviteCreateInput[] | MemberInviteUpdateInput[],
            onSuccess: (data) => { handleCompleted(data); },
            onCompleted: () => { props.setSubmitting(false); },
        });
    }, [disabled, existing, fetch, handleCompleted, isCreate, props, values]);

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
                title={t(isCreate ? "CreateInvites" : "UpdateInvites")}
            />
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                height: "100%",
                overflow: "hidden",
            }}>
                <Typography variant="h5" p={2}>{t("InvitesGoingTo")}</Typography>
                <Box sx={{
                    display: "flex",
                    flexDirection: "column",
                    flexGrow: 1,
                    margin: "auto",
                    overflowY: "auto",
                    width: "min(500px, 100vw)",
                }}>
                    <ObjectList
                        loading={false}
                        items={values}
                        keyPrefix="invite-list-item"
                        onAction={noop}
                        onClick={noop}
                    />
                </Box>
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
                <Box mt={4}>
                    <Typography variant="h6" p={2}>{t("MessageOptional")}</Typography>
                    <RichInputBase
                        disabled={values.length <= 0}
                        fullWidth
                        maxChars={4096}
                        minRows={1}
                        onChange={setMessage}
                        name="message"
                        sxs={{
                            root: {
                                background: palette.primary.main,
                                color: palette.primary.contrastText,
                                paddingBottom: 2,
                                maxHeight: "min(50vh, 500px)",
                                width: "-webkit-fill-available",
                                margin: "0",
                            },
                            bar: { borderRadius: 0 },
                            textArea: { paddingRight: 4, border: "none" },
                        }}
                        value={message}
                    />
                </Box>
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
            </Box>
        </MaybeLargeDialog>
    );
};

export const MemberInviteUpsert = ({
    invites,
    isCreate,
    isOpen,
    ...props
}: MemberInviteUpsertProps) => {

    return (
        <Formik
            enableReinitialize={true}
            initialValues={invites}
            onSubmit={noopSubmit}
            validate={async (values) => await validateMemberInviteValues(values, invites, isCreate)}
        >
            {(formik) => <MemberInviteForm
                disabled={false}
                existing={invites}
                handleUpdate={() => { }}
                isCreate={isCreate}
                isReadLoading={false}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
