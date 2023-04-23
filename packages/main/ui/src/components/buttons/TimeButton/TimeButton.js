import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { HistoryIcon as TimeIcon } from "@local/icons";
import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { TimeMenu } from "../../lists/TimeMenu/TimeMenu";
import { searchButtonStyle } from "../styles";
export const TimeButton = ({ setTimeFrame, timeFrame, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [timeAnchorEl, setTimeAnchorEl] = useState(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState("");
    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label, frame) => {
        setTimeAnchorEl(null);
        setTimeFrame(frame);
        if (label)
            setTimeFrameLabel(label === "All Time" ? "" : label);
    };
    return (_jsxs(_Fragment, { children: [_jsx(TimeMenu, { anchorEl: timeAnchorEl, onClose: handleTimeClose }), _jsx(Tooltip, { title: t("TimeCreated"), placement: "top", children: _jsxs(Box, { onClick: handleTimeOpen, sx: searchButtonStyle(palette), children: [_jsx(TimeIcon, { fill: palette.secondary.main }), _jsx(Typography, { variant: "body2", sx: { marginLeft: 0.5 }, children: timeFrameLabel })] }) })] }));
};
//# sourceMappingURL=TimeButton.js.map