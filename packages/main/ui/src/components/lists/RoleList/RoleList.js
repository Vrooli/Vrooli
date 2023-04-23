import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Chip, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo } from "react";
export const RoleList = ({ maxCharacters = 50, roles, sx, }) => {
    const { palette } = useTheme();
    const [chips, numTagsCutOff] = useMemo(() => {
        let charactersBeforeCutoff = maxCharacters;
        const chipResult = [];
        for (let i = 0; i < roles.length; i++) {
            const role = roles[i];
            if (role.name.length < charactersBeforeCutoff) {
                charactersBeforeCutoff -= role.name.length;
                chipResult.push(_jsx(Chip, { label: role.name, size: "small", sx: {
                        backgroundColor: palette.mode === "light" ? "#1d7691" : "#016d97",
                        color: "white",
                        width: "fit-content",
                    } }, role.name));
            }
        }
        const numTagsCutOff = roles.length - chipResult.length;
        return [chipResult, numTagsCutOff];
    }, [maxCharacters, palette.mode, roles]);
    return (_jsx(Tooltip, { title: roles.map(t => t.name).join(", "), placement: "top", children: _jsxs(Stack, { direction: "row", spacing: 1, justifyContent: "left", alignItems: "center", sx: { ...(sx ?? {}) }, children: [chips, numTagsCutOff > 0 && _jsxs(Typography, { variant: "body1", children: ["+", numTagsCutOff] })] }) }));
};
//# sourceMappingURL=RoleList.js.map