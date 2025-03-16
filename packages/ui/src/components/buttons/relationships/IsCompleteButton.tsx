import { Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { CompleteIcon as CIcon } from "../../../icons/common.js";
import { RelationshipButton, RelationshipChip, withRelationshipIcon } from "./styles.js";
import { IsCompleteButtonProps } from "./types.js";

const CompleteIcon = withRelationshipIcon(CIcon);

export function IsCompleteButton({
    isEditing,
}: IsCompleteButtonProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [field, , helpers] = useField("isComplete");

    const { Icon, label, tooltip } = useMemo(() => {
        const isComplete = field?.value;
        const iconColor = isEditing ? palette.primary.light : palette.secondary.contrastText;
        return {
            Icon: () => <CompleteIcon fill={iconColor} />,
            label: t(field?.value ? "Complete" : "Incomplete"),
            tooltip: t(`IsComplete${isComplete ? "True" : "False"}TogglePress${isEditing ? "Editable" : ""}`),
        };
    }, [field?.value, isEditing, palette.primary.light, palette.secondary.contrastText, t]);

    const handleClick = useCallback(() => {
        if (!isEditing) return;
        helpers.setValue(!field?.value);
    }, [isEditing, helpers, field?.value]);

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
