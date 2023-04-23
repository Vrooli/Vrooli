import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Card, CardContent, Typography, useTheme } from "@mui/material";
import { useDimensions } from "../../../utils/hooks/useDimensions";
import { LineGraph } from "../../graphs/LineGraph/LineGraph";
export const LineGraphCard = ({ title, index, ...lineGraphProps }) => {
    const { palette } = useTheme();
    const { dimensions, ref } = useDimensions();
    return (_jsx(Card, { ref: ref, sx: {
            width: "100%",
            height: "100%",
            boxShadow: 6,
            background: palette.primary.light,
            color: palette.primary.contrastText,
            borderRadius: "16px",
            margin: 0,
        }, children: _jsxs(CardContent, { sx: {
                display: "contents",
            }, children: [_jsx(Typography, { variant: "h6", component: "h2", textAlign: "center", sx: {
                        marginBottom: 2,
                        marginTop: 1,
                    }, children: title }), _jsx(LineGraph, { dims: {
                        width: dimensions.width,
                        height: 250,
                    }, ...lineGraphProps })] }) }));
};
//# sourceMappingURL=LineGraphCard.js.map