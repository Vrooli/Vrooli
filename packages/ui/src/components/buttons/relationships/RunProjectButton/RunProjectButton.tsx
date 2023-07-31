import { exists } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { buttonSx } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemRunProject } from "components/lists/types";
import { useField } from "formik";
import { AddIcon, ProjectIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { openObject } from "utils/navigation/openObject";
import { largeButtonProps } from "../styles";
import { RunProjectButtonProps } from "../types";

export function RunProjectButton({
    isEditing,
    objectType,
    zIndex,
}: RunProjectButtonProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [field, , helpers] = useField("runProject");

    const isAvailable = useMemo(() => ["Schedule"].includes(objectType) && ["boolean", "object"].includes(typeof field.value), [objectType, field.value]);

    // Project run dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const runProject = field?.value;
        // If not editing, navigate to project run
        if (!isEditing) {
            if (runProject) openObject(runProject, setLocation);
        }
        else {
            // If project run was set, remove
            if (runProject) {
                exists(helpers) && helpers.setValue(null);
            }
            // Otherwise, open select dialog
            else setDialogOpen(true);
        }
    }, [isAvailable, field?.value, isEditing, setLocation, helpers]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((runProject: RelationshipItemRunProject) => {
        const runProjectId = field?.value?.id;
        if (runProject?.id === runProjectId) return;
        exists(helpers) && helpers.setValue(runProject);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => unknown, () => unknown]>(() => {
        if (isDialogOpen) return ["RunProject", handleSelect, closeDialog];
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        return [null, () => { }, () => { }];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const runProject = field?.value;
        // If no data, marked as unset
        if (!runProject) return {
            Icon: AddIcon,
            tooltip: t(`RunNoneTogglePress${isEditing ? "Editable" : ""}`),
        };
        const runName = getDisplay(runProject).title;
        return {
            Icon: ProjectIcon,
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
                zIndex={zIndex + 1}
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
                            ...buttonSx(palette.primary.light, !isEditing),
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
