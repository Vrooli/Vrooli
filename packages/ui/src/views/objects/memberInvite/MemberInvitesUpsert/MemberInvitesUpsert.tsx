import { endpointPostMemberInvites, endpointPutMemberInvites, MemberInvite, MemberInviteCreateInput, MemberInviteUpdateInput, memberInviteValidation, noop, noopSubmit } from "@local/shared";
import { Box, Checkbox, FormControlLabel, Typography, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Field, Formik } from "formik";
import { useHistoryState } from "hooks/useHistoryState";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { MemberInviteShape, shapeMemberInvite } from "utils/shape/models/memberInvite";
import { MemberInvitesFormProps, MemberInvitesUpsertProps } from "../types";

// const memberInviteInitialValues = (
//     session: Session | undefined,
//     existing: NewMemberInviteShape,
// ): MemberInviteShape => ({
//     message: "",
//     willBeAdmin: false,
//     willHavePermissions: JSON.stringify({}),
//     ...existing,
//     user: {
//         __typename: "User" as const,
//         ...existing.user,
//         id: existing.user?.id ?? DUMMY_ID,
//     },
// });

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

const MemberInvitesForm = ({
    disabled,
    dirty,
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
}: MemberInvitesFormProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const { palette } = useTheme();
    const [message, setMessage] = useHistoryState("member-invite-message", "");

    const { handleCancel, handleCompleted } = useUpsertActions<MemberInvite[]>({
        display,
        isCreate,
        objectType: "MemberInvite",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<MemberInvite[], MemberInviteCreateInput[], MemberInviteUpdateInput[]>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostMemberInvites,
        endpointUpdate: endpointPutMemberInvites,
    });
    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<MemberInviteCreateInput[] | MemberInviteUpdateInput[], MemberInvite[]>({
        disabled,
        existing,
        fetch,
        inputs: transformMemberInviteValues(values, existing, isCreate) as MemberInviteCreateInput[] | MemberInviteUpdateInput[],
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="member-invite-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
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

export const MemberInvitesUpsert = ({
    invites,
    isCreate,
    isOpen,
    ...props
}: MemberInvitesUpsertProps) => {

    return (
        <Formik
            enableReinitialize={true}
            initialValues={invites}
            onSubmit={noopSubmit}
            validate={async (values) => await validateMemberInviteValues(values, invites, isCreate)}
        >
            {(formik) => <MemberInvitesForm
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
