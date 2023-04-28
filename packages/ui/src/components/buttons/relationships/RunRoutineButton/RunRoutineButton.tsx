import { exists, RoutineIcon, useLocation } from "@local/shared";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemRunRoutine } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { SessionContext } from "utils/SessionContext";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
import { RunRoutineButtonProps } from "../types";

export function RunRoutineButton({
    isEditing,
    objectType,
    zIndex,
}: RunRoutineButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [field, , helpers] = useField("runRoutine");

    const isAvailable = useMemo(() => ["Schedule"].includes(objectType) && exists(field.value), [objectType, field.value]);

    // Routine run dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<any>) => {
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
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => any, () => void]>(() => {
        if (isDialogOpen) return ["RunRoutine", handleSelect, closeDialog];
        return [null, () => { }, () => { }];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const runRoutine = field?.value;
        // If no data, marked as unset
        if (!runRoutine) return {
            Icon: null,
            tooltip: isEditing ? "" : "Press to assign to a routine run",
        };
        const runRoutineName = getDisplay(runRoutine).title;
        return {
            Icon: RoutineIcon,
            tooltip: `Routine run: ${runRoutineName}`,
        };
    }, [isEditing, languages, field?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    // Return button with label on top
    return (
        <>
            {/* Popup for selecting routine run */}
            {findType && <FindObjectDialog
                find="List"
                isOpen={Boolean(findType)}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={[findType]}
                zIndex={zIndex + 1}
            />}
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
            >
                <TextShrink id="runRoutine" sx={{ ...commonLabelProps() }}>
                    {t("RunRoutine", { count: 1 })}
                </TextShrink>
                <Tooltip title={tooltip}>
                    <ColorIconButton
                        background={palette.primary.light}
                        sx={{ ...commonButtonProps(isEditing, true) }}
                        onClick={handleClick}
                    >
                        {Icon && <Icon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>
            </Stack>
        </>
    );
}
