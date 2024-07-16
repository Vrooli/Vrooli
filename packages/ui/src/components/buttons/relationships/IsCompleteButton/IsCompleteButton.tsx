import { Tooltip } from "@mui/material";
import { useField } from "formik";
import { CompleteIcon as CIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { RelationshipButton, RelationshipChip, withRelationshipIcon } from "../styles";
import { IsCompleteButtonProps } from "../types";

const CompleteIcon = withRelationshipIcon(CIcon);

export function IsCompleteButton({
    isEditing,
}: IsCompleteButtonProps) {
    const { t } = useTranslation();

    const [field, , helpers] = useField("isComplete");

    const { Icon, label, tooltip } = useMemo(() => {
        const isComplete = field?.value;
        return {
            Icon: isComplete ? CompleteIcon : undefined,
            label: t(field?.value ? "Complete" : "Incomplete"),
            tooltip: t(`IsComplete${isComplete ? "True" : "False"}TogglePress${isEditing ? "Editable" : ""}`),
        };
    }, [field?.value, isEditing, t]);

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
