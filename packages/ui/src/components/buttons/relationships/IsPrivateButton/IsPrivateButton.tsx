import { Tooltip } from "@mui/material";
import { useField } from "formik";
import { InvisibleIcon, VisibleIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { RelationshipButton, RelationshipChip, withRelationshipIcon } from "../styles";
import { IsPrivateButtonProps } from "../types";

const PrivateIcon = withRelationshipIcon(InvisibleIcon);
const PublicIcon = withRelationshipIcon(VisibleIcon);

export function IsPrivateButton({
    isEditing,
}: IsPrivateButtonProps) {
    const { t } = useTranslation();

    const [versionField, , versionHelpers] = useField("isPrivate");
    const [, , rootHelpers] = useField("root.isPrivate");
    const [rootVersionsCountField] = useField("root.versionsCount");

    const { Icon, label, tooltip } = useMemo(() => {
        const isPrivate = versionField?.value;
        return {
            Icon: isPrivate ? PrivateIcon : PublicIcon,
            label: t(versionField?.value ? "Private" : "Public"),
            tooltip: t(`${!isPrivate ? "Private" : "Public"}TogglePress${isEditing ? "Editable" : ""}`),
        };
    }, [versionField?.value, isEditing, t]);

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
