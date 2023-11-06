import { exists, noop } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemRunRoutine } from "components/lists/types";
import { useField } from "formik";
import { AddIcon, RoutineIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { highlightStyle } from "styles";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { openObject } from "utils/navigation/openObject";
import { largeButtonProps } from "../styles";
import { RunRoutineButtonProps } from "../types";

export function RunRoutineButton({
    isEditing,
    objectType,
}: RunRoutineButtonProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [field, , helpers] = useField("runRoutine");

    const isAvailable = useMemo(() => ["Schedule"].includes(objectType) && ["boolean", "object"].includes(typeof field.value), [objectType, field.value]);

    // Routine run dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const runRoutine = field?.value;
        // If not editing, navigate to routine run
        if (!isEditing) {
            if (runRoutine) openObject(runRoutine, setLocation);
        }
        else {
            // If routine run was set, remove
            if (runRoutine) {
                exists(helpers) && helpers.setValue(null);
            }
            // Otherwise, open select dialog
            else setDialogOpen(true);
        }
    }, [isAvailable, field?.value, isEditing, setLocation, helpers]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((runRoutine: RelationshipItemRunRoutine) => {
        const runRoutineId = field?.value?.id;
        if (runRoutine?.id === runRoutineId) return;
        exists(helpers) && helpers.setValue(runRoutine);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => unknown, () => unknown]>(() => {
        if (isDialogOpen) return ["RunRoutine", handleSelect, closeDialog];
        return [null, noop, noop];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const runRoutine = field?.value;
        // If no data, marked as unset
        if (!runRoutine) return {
            Icon: AddIcon,
            tooltip: t(`RunNoneTogglePress${isEditing ? "Editable" : ""}`),
        };
        const runName = getDisplay(runRoutine).title;
        return {
            Icon: RoutineIcon,
            tooltip: t(`RunTogglePress${isEditing ? "Editable" : ""}`, { run: runName }),
        };
    }, [isEditing, field?.value, t]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    return (
        <>
            {/* Popup for selecting focus mode */}
            {findType && <FindObjectDialog
                find="List"
                isOpen={Boolean(findType)}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={[findType]}
            />}
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
                sx={{
                    marginTop: "auto",
                }}
            >
                <Tooltip title={tooltip}>
                    <Stack
                        direction="row"
                        justifyContent="center"
                        alignItems="center"
                        onClick={handleClick}
                        sx={{
                            borderRadius: 8,
                            paddingRight: 2,
                            ...largeButtonProps(isEditing, true),
                            ...highlightStyle(palette.primary.light, !isEditing),
                        }}
                    >
                        {Icon && (
                            <IconButton>
                                <Icon width={"48px"} height={"48px"} fill="white" />
                            </IconButton>
                        )}
                        <Typography variant="body1" sx={{ color: "white" }}>
                            {firstString(getDisplay(field?.value).title, t("Run", { count: 1 }))}
                        </Typography>
                    </Stack>
                </Tooltip>
            </Stack>
        </>
    );
}
