import { DeleteIcon, LINKS, Success, useLocation, UserDeleteInput, userDeleteOne, userDeleteOneSchema as validationSchema } from "@local/shared";
import { Button, Checkbox, DialogContent, FormControlLabel, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCustomMutation } from "api";
import { mutationWrapper } from "api/utils";
import { PasswordTextField } from "components/inputs/PasswordTextField/PasswordTextField";
import { Formik } from "formik";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { DeleteAccountDialogProps } from "../types";

const titleId = "delete-object-dialog-title";

/**
 * Dialog for deleting your account
 * @returns 
 */
export const DeleteAccountDialog = ({
    handleClose,
    isOpen,
    zIndex,
}: DeleteAccountDialogProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { id, name } = useMemo(() => getCurrentUser(session), [session]);
    const [deleteAccount] = useCustomMutation<Success, UserDeleteInput>(userDeleteOne);

    return (
        <LargeDialog
            id="delete-account-dialog"
            isOpen={isOpen}
            onClose={() => { handleClose(false); }}
            titleId={titleId}
            zIndex={zIndex}
        >
            <DialogTitle
                id={titleId}
                title={`Delete "${name}"`}
                onClose={() => { handleClose(false); }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    password: "",
                    deletePublicData: false,
                }}
                onSubmit={(values, helpers) => {
                    if (!id) {
                        PubSub.get().publishSnack({ messageKey: "NoUserIdFound", severity: "Error" });
                        return;
                    }
                    mutationWrapper<Success, UserDeleteInput>({
                        mutation: deleteAccount,
                        input: values,
                        successCondition: (data) => data.success,
                        successMessage: () => ({ messageKey: "AccountDeleteSuccess" }),
                        onSuccess: () => {
                            setLocation(LINKS.Home);
                            handleClose(true);
                        },
                        errorMessage: () => ({ messageKey: "AccountDeleteFail" }),
                        onError: () => {
                            handleClose(false);
                        },
                    });
                }}
                validationSchema={validationSchema}
            >
                {(formik) => <DialogContent>
                    <Stack direction="column" spacing={2} mt={2}>
                        <Typography variant="h6">Are you absolutely certain you want to delete the account of "{name}"?</Typography>
                        <Typography variant="h6" sx={{ color: palette.error.main, paddingBottom: 3 }}><b>This action cannot be undone.</b></Typography>
                        <Typography variant="h6">Enter your password to confirm.</Typography>
                        <PasswordTextField
                            fullWidth
                            name="password"
                            autoComplete="current-password"
                        />
                        {/* Is internal checkbox */}
                        <Tooltip placement={"top"} title="If checked, all public data owned by your account will be deleted. Please consider transferring and/or exporting anything important if you decide to choose this.">
                            <FormControlLabel
                                label='Delete public data'
                                control={
                                    <Checkbox
                                        id='delete-public-data'
                                        size="small"
                                        name='deletePublicData'
                                        color='secondary'
                                        checked={formik.values.deletePublicData}
                                        onChange={formik.handleChange}
                                    />
                                }
                            />
                        </Tooltip>
                        <Button
                            disabled={formik.isSubmitting || !formik.isValid}
                            startIcon={<DeleteIcon />}
                            color="secondary"
                            onClick={() => { formik.submitForm(); }}
                        >{t("Delete")}</Button>
                    </Stack>
                </DialogContent>}
            </Formik>
        </LargeDialog>
    );
};
