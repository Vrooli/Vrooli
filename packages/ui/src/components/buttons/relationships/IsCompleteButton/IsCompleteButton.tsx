import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { CompleteIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { IsCompleteButtonProps } from "../types";

export function IsCompleteButton({
    isEditing,
    objectType,
}: IsCompleteButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [field, , helpers] = useField("isComplete");

    const isAvailable = useMemo(() => ["Project", "Routine"].includes(objectType), [objectType]);

    const { Icon, tooltip } = useMemo(() => {
        const isComplete = field?.value;
        return {
            Icon: isComplete ? CompleteIcon : null,
            tooltip: t(`IsComplete${isComplete ? "True" : "False"}TogglePress${isEditing ? "Editable" : ""}`),
        };
    }, [field?.value, isEditing, t]);

    const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
        if (!isEditing || !isAvailable) return;
        helpers.setValue(!field?.value);
    }, [isEditing, isAvailable, helpers, field?.value]);

    // If not available, return null
    if (!isAvailable) return null;
    // Return button with label on top
    return (
        <Stack
            direction="column"
            alignItems="center"
            justifyContent="center"
        >
            <TextShrink id="complete" sx={{ ...commonLabelProps() }}>{t(field?.value ? "Complete" : "Incomplete")}</TextShrink>
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
}
