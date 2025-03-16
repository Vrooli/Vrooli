import { Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { InvisibleIcon, VisibleIcon } from "../../../icons/common.js";
import { RelationshipButton, RelationshipChip, withRelationshipIcon } from "./styles.js";
import { IsPrivateButtonProps } from "./types.js";

const PrivateIcon = withRelationshipIcon(InvisibleIcon);
const PublicIcon = withRelationshipIcon(VisibleIcon);

export function IsPrivateButton({
    isEditing,
}: IsPrivateButtonProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [versionField, , versionHelpers] = useField("isPrivate");
    const [, , rootHelpers] = useField("root.isPrivate");
    const [rootVersionsCountField] = useField("root.versionsCount");

    const { Icon, label, tooltip } = useMemo(() => {
        const isPrivate = versionField?.value;
        const iconColor = isEditing ? palette.primary.light : palette.secondary.contrastText;
        return {
            Icon: isPrivate ? () => <PrivateIcon fill={iconColor} /> : () => <PublicIcon fill={iconColor} />,
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
                    startIcon={Icon && <Icon />}
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
            icon={Icon && <Icon />}
            label={label}
        />
    );
}
