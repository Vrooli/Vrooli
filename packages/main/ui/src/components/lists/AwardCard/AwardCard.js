import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { Card, CardContent, LinearProgress, Typography, useTheme, } from "@mui/material";
import { useMemo } from "react";
export const AwardCard = ({ award, isEarned, }) => {
    const { palette } = useTheme();
    const { title, description, level } = useMemo(() => {
        if (!isEarned) {
            if (award.nextTier)
                return award.nextTier;
            if (award.earnedTier)
                return award.earnedTier;
        }
        else if (award.earnedTier)
            return award.earnedTier;
        return { title: "", description: "", level: 0 };
    }, [award.earnedTier, award.nextTier, isEarned]);
    const percentage = useMemo(() => {
        if (award.progress === 0)
            return 0;
        if (level === 0)
            return -1;
        return Math.round((award.progress / level) * 100);
    }, [award.progress, level]);
    return (_jsx(Card, { sx: {
            width: "100%",
            height: "100%",
            background: isEarned ? palette.secondary.main : palette.primary.light,
            color: palette.primary.contrastText,
            borderRadius: "16px",
            margin: 0,
        }, children: _jsxs(CardContent, { sx: {
                display: "flex",
                flexDirection: "column",
                justifyContent: "space-between",
                height: "100%",
            }, children: [_jsx(Typography, { variant: "h6", component: "h2", textAlign: "center", mb: 2, children: title }), _jsx(Typography, { variant: "body2", component: "p", textAlign: "center", mb: 4, children: description }), percentage >= 0 && _jsxs(_Fragment, { children: [_jsx(LinearProgress, { variant: "determinate", value: percentage, sx: {
                                margin: 2,
                                marginBottom: 1,
                                height: "12px",
                                borderRadius: "12px",
                            } }), _jsxs(Typography, { variant: "body2", component: "p", textAlign: "center", children: [award.progress, " / ", level, " (", percentage, "%)"] })] })] }) }));
};
//# sourceMappingURL=AwardCard.js.map