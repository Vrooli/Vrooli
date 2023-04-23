import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Chip, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo } from "react";
export const TagList = ({ maxCharacters = 50, parentId, sx, tags, }) => {
    const { palette } = useTheme();
    const [chips, numTagsCutOff] = useMemo(() => {
        let charactersBeforeCutoff = maxCharacters;
        const chipResult = [];
        for (let i = 0; i < tags.length; i++) {
            const tag = tags[i];
            if (tag?.tag && tag.tag.length < charactersBeforeCutoff) {
                charactersBeforeCutoff -= tag.tag.length;
                chipResult.push(_jsx(Chip, { label: tag.tag, size: "small", sx: {
                        backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0",
                        color: "white",
                        width: "fit-content",
                    } }, tag.tag));
            }
        }
        const numTagsCutOff = tags.length - chipResult.length;
        return [chipResult, numTagsCutOff];
    }, [maxCharacters, palette.mode, tags]);
    return (_jsx(Tooltip, { title: tags.map(t => t.tag).join(", "), placement: "top", children: _jsxs(Stack, { direction: "row", spacing: 1, justifyContent: "left", alignItems: "center", sx: { ...(sx ?? {}) }, children: [chips, numTagsCutOff > 0 && _jsxs(Typography, { variant: "body1", children: ["+", numTagsCutOff] })] }) }));
};
//# sourceMappingURL=TagList.js.map