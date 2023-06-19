import { AddIcon, CompleteIcon } from "@local/shared";
import { Button, Checkbox, FormControlLabel, Grid, Tooltip, Typography } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { SessionContext } from "utils/SessionContext";
import { StandardInput } from "../standards/StandardInput/StandardInput";
import { StandardVersionSelectSwitchProps } from "../types";

export function StandardVersionSelectSwitch({
    canUpdateStandardVersion,
    selected,
    onChange,
    disabled,
    zIndex,
}: StandardVersionSelectSwitchProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    console.log("StandardVersionSelectSwitch", selected, onChange, disabled, zIndex);

    const [usingStandard, setUsingStandard] = useState<boolean>(selected !== null);

    // Create dialog
    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState<boolean>(false);
    const openCreateDialog = useCallback(() => { setIsCreateDialogOpen(true); }, [setIsCreateDialogOpen]);
    const closeCreateDialog = useCallback(() => { setIsCreateDialogOpen(false); }, [setIsCreateDialogOpen]);

    const [isConnecting, setIsConnecting] = useState<boolean | null>(null);
    const handleConnectClick = useCallback(() => {
        setIsConnecting(true);
        openCreateDialog();
        // Remove create data, if any
        onChange(null);
    }, []);
    const handleCreateClick = useCallback(() => {
        setIsConnecting(false);
        closeCreateDialog();
        // Remove connect data, if any
        onChange(null);
    }, []);

    return (
        <>
            {/* Popup for adding/connecting a new standardVersion */}
            <FindObjectDialog
                find="Full"
                isOpen={isCreateDialogOpen}
                handleComplete={onChange as any}
                handleCancel={closeCreateDialog}
                limitTo={["StandardVersion"]}
                zIndex={zIndex + 1}
            />
            {/* Main component */}
            <Grid container spacing={1}>
                <Grid item xs={12} md={6}>
                    <Tooltip placement={"right"} title='Should this be in a specific format?'>
                        <FormControlLabel
                            disabled={disabled}
                            label='Use standard'
                            control={
                                <Checkbox
                                    size="small"
                                    name='usingStandard'
                                    color='secondary'
                                    checked={usingStandard}
                                    onChange={(e) => setUsingStandard(e.target.checked)}
                                />
                            }
                        />
                    </Tooltip>
                </Grid>
                {usingStandard && (
                    <>
                        <Grid item xs={6} md={3}>
                            <Button
                                fullWidth
                                color="secondary"
                                onClick={handleConnectClick}
                                variant="outlined"
                                startIcon={isConnecting === true ? <CompleteIcon /> : undefined}
                            >{t("Connect")}</Button>
                        </Grid>
                        <Grid item xs={6} md={3}>
                            <Button
                                fullWidth
                                color="secondary"
                                onClick={handleCreateClick}
                                variant="outlined"
                                startIcon={isConnecting === false ? <CompleteIcon /> : undefined}
                            >{t("Create")}</Button>
                        </Grid>
                    </>
                )}
                <Grid item xs={12}>
                    <Typography variant="h6" sx={{ ...noSelect }}>{selected ? getTranslation(selected, getUserLanguages(session)).name : t("Custom")}</Typography>
                </Grid>
                {(selected && isConnecting === false) && (
                    <Grid item xs={12}>
                        <StandardInput
                            disabled={!canUpdateStandardVersion}
                            fieldName="preview"
                            zIndex={zIndex}
                        />
                    </Grid>
                )}
                {/* Show button to open connect dialog, if closed without selecting a standard */}
                {!selected && isConnecting === true && (
                    <Button
                        fullWidth
                        color="secondary"
                        onClick={handleCreateClick}
                        variant="contained"
                        startIcon={<AddIcon />}
                    >{t("Add")}</Button>
                )}
            </Grid>
        </>
    );
}
