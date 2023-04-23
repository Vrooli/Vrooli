import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CommentIcon } from "@local/icons";
import { Box, ListItemText, Stack, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { multiLineEllipsis } from "../../../styles";
import { getObjectUrl } from "../../../utils/navigation/openObject";
import { useLocation } from "../../../utils/route";
export const CommentsButton = ({ commentsCount = 0, disabled = false, object, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const link = useMemo(() => object ? `${getObjectUrl(object)}#comments` : "", [object]);
    const handleClick = useCallback((event) => {
        event.stopPropagation();
        event.preventDefault();
        if (link.length === 0)
            return;
        setLocation(link);
    }, [link, setLocation]);
    return (_jsxs(Stack, { direction: "row", spacing: 0.5, sx: {
            marginRight: 0,
            pointerEvents: "none",
        }, children: [_jsx(Box, { component: "a", href: link, onClick: handleClick, sx: {
                    display: "contents",
                    cursor: disabled ? "none" : "pointer",
                    pointerEvents: disabled ? "none" : "all",
                }, children: _jsx(CommentIcon, { fill: disabled ? "rgb(189 189 189)" : palette.secondary.main }) }), _jsx(ListItemText, { primary: commentsCount, sx: { ...multiLineEllipsis(1), pointerEvents: "none" } })] }));
};
//# sourceMappingURL=CommentsButton.js.map