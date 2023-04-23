import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CompleteIcon } from "@local/icons";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { TextShrink } from "../../../text/TextShrink/TextShrink";
import { ColorIconButton } from "../../ColorIconButton/ColorIconButton";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
export function IsCompleteButton({ isEditing, objectType, }) {
    const { palette } = useTheme();
    const [field, , helpers] = useField("isComplete");
    const isAvailable = useMemo(() => ["Project", "Routine"].includes(objectType), [objectType]);
    const { Icon, tooltip } = useMemo(() => {
        const isComplete = field?.value;
        return {
            Icon: isComplete ? CompleteIcon : null,
            tooltip: isComplete ? `This is complete${isEditing ? "" : ". Press to mark incomplete"}` : `This is incomplete${isEditing ? "" : ". Press to mark complete"}`,
        };
    }, [field?.value, isEditing]);
    const handleClick = useCallback((ev) => {
        if (!isEditing || !isAvailable)
            return;
        helpers.setValue(!field?.value);
    }, [isEditing, isAvailable, helpers, field?.value]);
    if (!isAvailable)
        return null;
    return (_jsxs(Stack, { direction: "column", alignItems: "center", justifyContent: "center", children: [_jsx(TextShrink, { id: "complete", sx: { ...commonLabelProps() }, children: "Complete?" }), _jsx(Tooltip, { title: tooltip, children: _jsx(ColorIconButton, { background: palette.primary.light, sx: { ...commonButtonProps(isEditing, false) }, onClick: handleClick, children: Icon && _jsx(Icon, { ...commonIconProps() }) }) })] }));
}
//# sourceMappingURL=IsCompleteButton.js.map