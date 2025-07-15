import Box from "@mui/material/Box";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { memberInviteFormConfig, noop, noopSubmit, validateAndGetYupErrors, type MemberInvite, type MemberInviteCreateInput, type MemberInviteShape, type MemberInviteUpdateInput } from "@vrooli/shared";
import { Field, Formik } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import Dialog from "@mui/material/Dialog";
import { useIsMobile } from "../../../hooks/useIsMobile.js";
import { AdvancedInputBase } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { ObjectList } from "../../../components/lists/ObjectList/ObjectList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { useHistoryState } from "../../../hooks/useHistoryState.js";
import { useStandardBatchUpsertForm } from "../../../hooks/useStandardBatchUpsertForm.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { type MemberInvitesFormProps, type MemberInvitesUpsertProps } from "./types.js";

export function transformMemberInviteValues(values: MemberInviteShape[], existing: MemberInviteShape[], isCreate: boolean) {
    return isCreate ?
        values.map((value) => memberInviteFormConfig.transformations.shapeToInput.create?.(value)) :
        values.map((value, index) => memberInviteFormConfig.transformations.shapeToInput.update?.(existing[index], value)); // Assumes the dialog doesn't change the order or remove items
}

async function validateMemberInviteValues(values: MemberInviteShape[], existing: MemberInviteShape[], isCreate: boolean) {
    const transformedValues = transformMemberInviteValues(values, existing, isCreate);
    const validationSchema = memberInviteFormConfig.validation.schema[isCreate ? "create" : "update"]({ env: process.env.NODE_ENV });
    const result = await Promise.all(transformedValues.map(async (value) => await validateAndGetYupErrors(validationSchema, value)));

    // Filter and combine the result into one object with only error results
    const combinedResult = result.reduce((acc, curr, index) => {
        if (Object.keys(curr).length > 0) {  // check if the object has any keys (errors)
            acc[index] = curr;
        }
        return acc;
    }, {} as any);

    return combinedResult;
}

function MemberInvitesForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isMutate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: MemberInvitesFormProps) {
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const [message, setMessage] = useHistoryState("member-invite-message", "");
    const isMobile = useIsMobile();
    const isMobileOld = useWindowSize(({ width }) => width <= breakpoints.values.md);

    // Use the standardized batch form hook
    const {
        isLoading,
        handleCancel,
        handleCompleted,
        onSubmit,
    } = useStandardBatchUpsertForm({
        objectType: "MemberInvite",
        transformFunction: transformMemberInviteValues,
        validateFunction: validateMemberInviteValues,
        endpoints: {
            create: memberInviteFormConfig.endpoints.createOne,
            update: memberInviteFormConfig.endpoints.updateOne,
        },
    }, {
        values,
        existing,
        isCreate,
        display,
        disabled,
        isMutate,
        isReadLoading,
        isSubmitting: props.isSubmitting,
        handleUpdate,
        setSubmitting: props.setSubmitting,
        onCancel,
        onCompleted,
        onDeleted,
        onClose,
    });

    const handleSubmit = useCallback(() => {
        setMessage("");
        onSubmit();
    }, [setMessage, onSubmit]);

    const [inputFocused, setInputFocused] = useState(false);
    const onFocus = useCallback(() => { setInputFocused(true); }, []);
    const onBlur = useCallback(() => { setInputFocused(false); }, []);

    const dialogContent = (
        <>
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
                <Typography variant="h6" p={2}>{t("InvitesGoingTo")}</Typography>
                <Box sx={{
                    display: "flex",
                    flexDirection: "column",
                    flexGrow: 1,
                    margin: "auto",
                    overflowY: "auto",
                    width: "min(500px, 100vw)",
                    pointerEvents: "none",
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
                    <Typography variant="h6" p={2}>{t("Message", { count: 1 })}</Typography>
                    <AdvancedInputBase
                        disabled={values.length <= 0}
                        features={{ maxChars: 4096, maxRowsCollapsed: 2, maxRowsExpanded: 10, minRowsCollapsed: 1 }}
                        onBlur={onBlur}
                        onChange={setMessage}
                        onFocus={onFocus}
                        name="message"
                        sxs={{
                            root: {
                                background: palette.primary.main,
                                color: palette.primary.contrastText,
                                paddingBottom: 2,
                                maxHeight: "min(75vh, 500px)",
                                width: "-webkit-fill-available",
                                margin: "0",
                            },
                            topBar: { borderRadius: 0, paddingLeft: isMobileOld ? "20px" : 0, paddingRight: isMobileOld ? "20px" : 0 },
                            bottomBar: { paddingLeft: isMobileOld ? "20px" : 0, paddingRight: isMobileOld ? "20px" : 0 },
                            inputRoot: {
                                border: "none",
                                background: palette.background.paper,
                            },
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
                    onSubmit={handleSubmit}
                />
            </Box>
        </>
    );

    return isMobile ? (
        <Dialog
            id="member-invite-upsert-dialog"
            open={isOpen}
            onClose={onClose}
            fullScreen
        >
            {dialogContent}
        </Dialog>
    ) : (
        <Dialog
            id="member-invite-upsert-dialog"
            open={isOpen}
            onClose={onClose}
            maxWidth="sm"
            fullWidth
        >
            {dialogContent}
        </Dialog>
    );
}

export function MemberInvitesUpsert({
    invites,
    isCreate,
    isMutate,
    isOpen,
    ...props
}: MemberInvitesUpsertProps) {

    async function validateValues(values: MemberInviteShape[]) {
        return await validateMemberInviteValues(values, invites, isCreate);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={invites}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <MemberInvitesForm
                disabled={false}
                existing={invites}
                handleUpdate={() => { }}
                isCreate={isCreate}
                isMutate={isMutate}
                isReadLoading={false}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
