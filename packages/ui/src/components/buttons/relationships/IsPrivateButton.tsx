import { Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Icon } from "../../../icons/Icons.js";
import { RelationshipButton, RelationshipChip } from "./styles.js";
import { IsPrivateButtonProps } from "./types.js";

export function IsPrivateButton({
    isEditing,
}: IsPrivateButtonProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [versionField, , versionHelpers] = useField("isPrivate");
    const [, , rootHelpers] = useField("root.isPrivate");
    const [rootVersionsCountField] = useField("root.versionsCount");

    const { iconColor, iconInfo, label, tooltip } = useMemo(() => {
        const isPrivate = versionField?.value;
        return {
            iconColor: isEditing ? palette.primary.light : palette.secondary.contrastText,
            iconInfo: { name: isPrivate ? "Invisible" : "Visible", type: "Common" } as const,
            label: t(versionField?.value ? "Private" : "Public"),
            tooltip: t(`${!isPrivate ? "Private" : "Public"}TogglePress${isEditing ? "Editable" : ""}`),
        };
    }, [versionField?.value, isEditing, palette.primary.light, palette.secondary.contrastText, t]);

    const handleClick = useCallback(() => {
        if (!isEditing) return;
        const updatedValue = !versionField?.value;
        versionHelpers.setValue(updatedValue);
        // If there is only one version, set root.isPrivate to the same value
        if (!Number.isNaN(rootVersionsCountField.value) && rootVersionsCountField.value === 1) {
            rootHelpers.setValue(updatedValue);
        }
    }, [isEditing, rootHelpers, rootVersionsCountField?.value, versionHelpers, versionField?.value]);

    // If editing, return button
    if (isEditing) {
        return (
            <Tooltip title={tooltip}>
                <RelationshipButton
                    onClick={handleClick}
                    startIcon={iconInfo && <Icon
                        fill={iconColor}
                        info={iconInfo}
                        size={24}
                    />}
                    variant="outlined"
                >
                    {label}
                </RelationshipButton>
            </Tooltip>
        );
    }
    // Otherwise, return chip
    return (
        <RelationshipChip
            icon={iconInfo && <Icon
                fill={iconColor}
                info={iconInfo}
                size={24}
            />}
            label={label}
        />
    );
}
