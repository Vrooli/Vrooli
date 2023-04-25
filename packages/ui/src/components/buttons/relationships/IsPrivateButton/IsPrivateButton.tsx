import { InvisibleIcon, VisibleIcon } from "@local/shared";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
import { IsPrivateButtonProps } from "../types";

export function IsPrivateButton({
    isEditing,
}: IsPrivateButtonProps) {
    const { palette } = useTheme();

    const [field, , helpers] = useField("isPrivate");

    const isAvailable = true;

    const { Icon, tooltip } = useMemo(() => {
        const isPrivate = field?.value;
        return {
            Icon: isPrivate ? InvisibleIcon : VisibleIcon,
            tooltip: isPrivate ? `Only you or your organization can see this${isEditing ? "" : ". Press to make public"}` : `Anyone can see this${isEditing ? "" : ". Press to make private"}`,
        };
    }, [field?.value, isEditing]);

    const handleClick = useCallback((ev: React.MouseEvent<any>) => {
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
            <TextShrink id="privacy" sx={{ ...commonLabelProps() }}>Privacy</TextShrink>
            <Tooltip title={tooltip}>
                <ColorIconButton
                    background={palette.primary.light}
                    sx={{ ...commonButtonProps(isEditing, false) }}
                    onClick={handleClick}
                >
                    {Icon && <Icon {...commonIconProps()} />}
                </ColorIconButton>
            </Tooltip>
        </Stack>
    );
}
