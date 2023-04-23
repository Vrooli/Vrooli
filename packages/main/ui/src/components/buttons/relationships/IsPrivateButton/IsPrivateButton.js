import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { InvisibleIcon, VisibleIcon } from "@local/icons";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { TextShrink } from "../../../text/TextShrink/TextShrink";
import { ColorIconButton } from "../../ColorIconButton/ColorIconButton";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
export function IsPrivateButton({ isEditing, }) {
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
    const handleClick = useCallback((ev) => {
        if (!isEditing || !isAvailable)
            return;
        helpers.setValue(!field?.value);
    }, [isEditing, isAvailable, helpers, field?.value]);
    if (!isAvailable)
        return null;
    return (_jsxs(Stack, { direction: "column", alignItems: "center", justifyContent: "center", children: [_jsx(TextShrink, { id: "privacy", sx: { ...commonLabelProps() }, children: "Privacy" }), _jsx(Tooltip, { title: tooltip, children: _jsx(ColorIconButton, { background: palette.primary.light, sx: { ...commonButtonProps(isEditing, false) }, onClick: handleClick, children: Icon && _jsx(Icon, { ...commonIconProps() }) }) })] }));
}
//# sourceMappingURL=IsPrivateButton.js.map