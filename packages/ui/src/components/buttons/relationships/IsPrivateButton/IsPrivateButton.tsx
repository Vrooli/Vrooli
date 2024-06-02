import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { InvisibleIcon, VisibleIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { IsPrivateButtonProps } from "../types";

export const IsPrivateButton = ({
    isEditing,
    objectType,
}: IsPrivateButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [versionField, , versionHelpers] = useField("isPrivate");
    const [, , rootHelpers] = useField("root.isPrivate");
    const [rootVersionsCountField] = useField("root.versionsCount");

    const isAvailable = useMemo(() => ["Api", "Code", "Note", "Project", "Routine", "RunProject", "RunRoutine", "Standard", "Team", "User"].includes(objectType), [objectType]);

    const { Icon, tooltip } = useMemo(() => {
        const isPrivate = versionField?.value;
        return {
            Icon: isPrivate ? InvisibleIcon : VisibleIcon,
            tooltip: t(`${!isPrivate ? "Private" : "Public"}TogglePress${isEditing ? "Editable" : ""}`),
        };
    }, [versionField?.value, isEditing, t]);

    const handleClick = useCallback(() => {
        if (!isEditing || !isAvailable) return;
        const updatedValue = !versionField?.value;
        versionHelpers.setValue(updatedValue);
        // If there is only one version, set root.isPrivate to the same value
        if (!Number.isNaN(rootVersionsCountField.value) && rootVersionsCountField.value === 1) {
            rootHelpers.setValue(updatedValue);
        }
    }, [isEditing, isAvailable, rootHelpers, rootVersionsCountField?.value, versionHelpers, versionField?.value]);

    // If not available, return null
    if (!isAvailable) return null;
    // Return button with label on top
    return (
        <Stack
            direction="column"
            alignItems="center"
            justifyContent="center"
        >
            <TextShrink id="privacy" sx={{ ...commonLabelProps() }}>{t(versionField?.value ? "Private" : "Public")}</TextShrink>
            <Tooltip title={tooltip}>
                <IconButton
                    onClick={handleClick}
                    sx={{ ...smallButtonProps(isEditing, false), background: palette.primary.light }}
                >
                    {Icon && <Icon {...commonIconProps()} />}
                </IconButton>
            </Tooltip>
        </Stack>
    );
};
