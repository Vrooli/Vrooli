import { Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Icon } from "../../../icons/Icons.js";
import { RelationshipButton, RelationshipChip } from "./styles.js";
import { IsCompleteButtonProps } from "./types.js";

export function IsCompleteButton({
    isEditing,
}: IsCompleteButtonProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [field, , helpers] = useField("isComplete");

    const { iconColor, iconInfo, label, tooltip } = useMemo(() => {
        const isComplete = field?.value;
        return {
            iconColor: isEditing ? palette.primary.light : palette.secondary.contrastText,
            iconInfo: { name: "Complete", type: "Common" } as const,
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
                    startIcon={iconInfo && <Icon
                        fill={iconColor}
                        info={iconInfo}
                        size={32}
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
                size={32}
            />}
            label={label}
        />
    );
}
