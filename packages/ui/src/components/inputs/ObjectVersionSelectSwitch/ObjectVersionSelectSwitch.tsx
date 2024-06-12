import { Button, Checkbox, FormControlLabel, Grid, Tooltip, Typography } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SessionContext } from "contexts/SessionContext";
import { CompleteIcon } from "icons";
import { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { StandardInput } from "../standards/StandardInput/StandardInput";
import { ObjectVersionSelectPayloads, ObjectVersionSelectSwitchProps } from "../types";

/**
 * Simple component for optionally connecting or creating a versioned object to some other object 
 */
export const ObjectVersionSelectSwitch = <T extends keyof ObjectVersionSelectPayloads>({
    canUpdate,
    disabled,
    label,
    selected,
    objectType,
    onChange,
    tooltip,
}: ObjectVersionSelectSwitchProps<T>) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const [usingObject, setusingObject] = useState<boolean>(selected !== null);

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
    }, [onChange, openCreateDialog]);
    const handleCreateClick = useCallback(() => {
        setIsConnecting(false);
        closeCreateDialog();
        // Remove connect data, if any
        onChange(null);
    }, [onChange, closeCreateDialog]);

    return (
        <>
            {/* Popup for adding/connecting a new object */}
            <FindObjectDialog
                find="Full"
                isOpen={isCreateDialogOpen}
                handleComplete={onChange as (value: ObjectVersionSelectPayloads[T]) => unknown}
                handleCancel={closeCreateDialog}
                limitTo={[objectType]}
                onlyVersioned={true}
            />
            {/* Main component */}
            <Grid container spacing={1}>
                <Grid item xs={12} md={6}>
                    <Tooltip placement={"right"} title={tooltip}>
                        <FormControlLabel
                            disabled={disabled}
                            label={label}
                            control={
                                <Checkbox
                                    size="small"
                                    name='usingObject'
                                    color='secondary'
                                    checked={usingObject}
                                    onChange={(e) => setusingObject(e.target.checked)}
                                />
                            }
                        />
                    </Tooltip>
                </Grid>
                {usingObject && (
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
                    <Typography variant="h6" sx={{ ...noSelect }}>{selected ? getTranslation<any>(selected, getUserLanguages(session)).name : t("Custom")}</Typography>
                </Grid>
                {(selected && isConnecting === false) && (
                    <Grid item xs={12}>
                        <StandardInput
                            disabled={!canUpdate}
                            fieldName="preview"
                        />
                    </Grid>
                )}
                {/* Show button to open connect dialog, if closed without selecting an object */}
                {/* {!selected && isConnecting === true && (
                    <Button
                        fullWidth
                        color="secondary"
                        onClick={handleCreateClick}
                        variant="contained"
                        startIcon={<AddIcon />}
                    >{t("Add")}</Button>
                )} */}
            </Grid>
        </>
    );
}
