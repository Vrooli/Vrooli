import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import Stack from "@mui/material/Stack";
import { Tooltip } from "../../Tooltip/Tooltip.js";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { LINKS, endpointsActions, userDeleteOneSchema as validationSchema, type DeleteAccountInput, type Session } from "@vrooli/shared";
import { Formik } from "formik";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SessionContext } from "../../../contexts/session.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { PubSub } from "../../../utils/pubsub.js";
import { PasswordTextInput } from "../../inputs/PasswordTextInput/PasswordTextInput.js";
import { Dialog, DialogContent, DialogActions } from "../Dialog/Dialog.js";
import { type DeleteAccountDialogProps } from "../types.js";

const initialValues = {
    password: "",
    deletePublicData: false,
} as const;

/**
 * Dialog for deleting your account
 * @returns 
 */
export function DeleteAccountDialog({
    handleClose,
    isOpen,
}: DeleteAccountDialogProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { id, name } = useMemo(() => getCurrentUser(session), [session]);
    const [deleteAccount] = useLazyFetch<DeleteAccountInput, Session>(endpointsActions.deleteAccount);

    function onClose() {
        handleClose(false);
    }

    function onSubmit(values: DeleteAccountInput) {
        if (!id) {
            PubSub.get().publish("snack", { messageKey: "NoUserIdFound", severity: "Error" });
            return;
        }
        fetchLazyWrapper<DeleteAccountInput, Session>({
            fetch: deleteAccount,
            inputs: values,
            successMessage: () => ({ messageKey: "AccountDeleteSuccess" }),
            onSuccess: (data) => {
                PubSub.get().publish("session", data);
                setLocation(LINKS.Home);
                handleClose(true);
            },
            errorMessage: () => ({ messageKey: "AccountDeleteFail" }),
            onError: () => {
                handleClose(false);
            },
        });
    }

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            title={`Delete "${name}"`}
            size="md"
            variant="danger"
        >
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={onSubmit}
                validationSchema={validationSchema}
            >
                {(formik) => <>
                    <DialogContent>
                        <Stack direction="column" spacing={2}>
                            <Typography variant="h6">Are you absolutely certain you want to delete the account of &quot;{name}&quot;?</Typography>
                            <Typography variant="h6" sx={{ color: palette.error.main, paddingBottom: 3 }}><b>This action cannot be undone.</b></Typography>
                            <Typography variant="h6">Enter your password to confirm.</Typography>
                            <PasswordTextInput
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
                        </Stack>
                    </DialogContent>
                    <DialogActions variant="danger">
                        <Button
                            onClick={onClose}
                            variant="text"
                        >{t("Cancel")}</Button>
                        <Button
                            disabled={formik.isSubmitting || !formik.isValid}
                            startIcon={<IconCommon
                                decorative
                                name="Delete"
                            />}
                            color="secondary"
                            onClick={() => { formik.submitForm(); }}
                            variant="contained"
                        >{t("Delete")}</Button>
                    </DialogActions>
                </>}
            </Formik>
        </Dialog>
    );
}
