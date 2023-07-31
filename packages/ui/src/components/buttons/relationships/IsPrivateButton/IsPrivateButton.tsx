import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { InvisibleIcon, VisibleIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { IsPrivateButtonProps } from "../types";

export function IsPrivateButton({
    isEditing,
    objectType,
}: IsPrivateButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [field, , helpers] = useField("isPrivate");

    const isAvailable = useMemo(() => ["Api", "Note", "Organization", "Project", "Routine", "RunProject", "RunRoutine", "SmartContract", "Standard", "User"].includes(objectType), [objectType]);

    const { Icon, tooltip } = useMemo(() => {
        const isPrivate = field?.value;
        return {
            Icon: isPrivate ? InvisibleIcon : VisibleIcon,
            tooltip: t(`${!isPrivate ? "Private" : "Public"}TogglePress${isEditing ? "Editable" : ""}`),
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
            <TextShrink id="privacy" sx={{ ...commonLabelProps() }}>{t(field?.value ? "Private" : "Public")}</TextShrink>
            <Tooltip title={tooltip}>
                <ColorIconButton
                    background={palette.primary.light}
                    sx={{ ...smallButtonProps(isEditing, false) }}
                    onClick={handleClick}
                >
                    {Icon && <Icon {...commonIconProps()} />}
                </ColorIconButton>
            </Tooltip>
        </Stack>
    );
}
