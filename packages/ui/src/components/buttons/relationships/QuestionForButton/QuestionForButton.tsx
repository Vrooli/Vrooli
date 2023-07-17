import { AddIcon, ApiIcon, exists, NoteIcon, OrganizationIcon, ProjectIcon, QuestionForType, RoutineIcon, SmartContractIcon, StandardIcon, SvgComponent } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { buttonSx } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemQuestionForObject } from "components/lists/types";
import { useField } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { openObject } from "utils/navigation/openObject";
import { largeButtonProps } from "../styles";
import { QuestionForButtonProps } from "../types";

/**
 * Maps QuestionForTypes to their icons
 */
const questionForTypeIcons: Record<QuestionForType, SvgComponent> = {
    Api: ApiIcon,
    Note: NoteIcon,
    Organization: OrganizationIcon,
    Project: ProjectIcon,
    Routine: RoutineIcon,
    SmartContract: SmartContractIcon,
    Standard: StandardIcon,
};

export function QuestionForButton({
    isEditing,
    objectType,
    zIndex,
}: QuestionForButtonProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [field, , helpers] = useField("forObject");

    const isAvailable = useMemo(() => ["Question"].includes(objectType) && ["boolean", "object"].includes(typeof field.value), [objectType, field.value]);

    // Select object dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const forObject = field?.value;
        // If not editing, navigate to object's page
        if (!isEditing) {
            if (forObject) openObject(forObject, setLocation);
        }
        else {
            // If focus mode was set, remove
            if (forObject) {
                exists(helpers) && helpers.setValue(null);
            }
            // Otherwise, open select dialog
            else setDialogOpen(true);
        }
    }, [isAvailable, field?.value, isEditing, setLocation, helpers]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((forObject: RelationshipItemQuestionForObject) => {
        const forObjectId = field?.value?.id;
        if (forObject?.id === forObjectId) return;
        exists(helpers) && helpers.setValue(forObject);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    // FindObjectDialog
    const [findHandleAdd, findHandleClose] = useMemo<[(item: any) => any, () => void]>(() => {
        if (isDialogOpen) return [handleSelect, closeDialog];
        return [() => { }, () => { }];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const forObject = field?.value;
        // If no data, marked as unset
        if (!forObject) return {
            Icon: AddIcon,
            tooltip: isEditing ? "" : "Press to link an object",
        };
        const forObjectName = getDisplay(forObject).title ?? "";
        return {
            Icon: questionForTypeIcons[forObject.__typename],
            tooltip: `For: ${forObjectName}`,
        };
    }, [isEditing, field?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    return (
        <>
            {/* Popup for selecting focus mode */}
            <FindObjectDialog
                find="List"
                isOpen={isDialogOpen}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={Object.keys(QuestionForType) as SelectOrCreateObjectType[]}
                zIndex={zIndex + 1}
            />
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
                        <IconButton>
                            <Icon width={"48px"} height={"48px"} fill="white" />
                        </IconButton>
                        <Typography variant="body1" sx={{ color: "white" }}>
                            {field?.value ? getDisplay(field?.value).title : "Question for..."}
                        </Typography>
                    </Stack>
                </Tooltip>
            </Stack>
        </>
    );
}
