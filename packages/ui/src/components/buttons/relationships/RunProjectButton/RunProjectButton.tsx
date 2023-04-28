import { exists, ProjectIcon, useLocation } from "@local/shared";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemRunProject } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { SessionContext } from "utils/SessionContext";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
import { RunProjectButtonProps } from "../types";

export function RunProjectButton({
    isEditing,
    objectType,
    zIndex,
}: RunProjectButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [field, , helpers] = useField("runProject");

    const isAvailable = useMemo(() => ["Schedule"].includes(objectType) && exists(field.value), [objectType, field.value]);

    // Project run dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<any>) => {
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
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => any, () => void]>(() => {
        if (isDialogOpen) return ["RunProject", handleSelect, closeDialog];
        return [null, () => { }, () => { }];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const runProject = field?.value;
        // If no data, marked as unset
        if (!runProject) return {
            Icon: null,
            tooltip: isEditing ? "" : "Press to assign to a project run",
        };
        const runProjectName = getDisplay(runProject).title;
        return {
            Icon: ProjectIcon,
            tooltip: `Project Run: ${runProjectName}`,
        };
    }, [isEditing, languages, field?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    // Return button with label on top
    return (
        <>
            {/* Popup for selecting project run */}
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
                <TextShrink id="runProject" sx={{ ...commonLabelProps() }}>
                    {t("RunProject", { count: 1 })}
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
