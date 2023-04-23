import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { EllipsisIcon, HelpIcon } from "@local/icons";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { reminderFindOne } from "../../../api/generated/endpoints/reminder_findOne";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { placeholderColor } from "../../../utils/display/listTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { useLocation } from "../../../utils/route";
export const ReminderView = ({ display = "page", partialData, zIndex = 200, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    const { id, isLoading, object: reminder, permissions, setObject: setReminder } = useObjectFromUrl({
        query: reminderFindOne,
        partialData,
    });
    const [moreMenuAnchor, setMoreMenuAnchor] = useState(null);
    const openMoreMenu = useCallback((ev) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);
    const actionData = useObjectActions({
        object: reminder,
        objectType: "Reminder",
        setLocation,
        setObject: setReminder,
    });
    const overviewComponent = useMemo(() => (_jsxs(Box, { position: "relative", ml: 'auto', mr: 'auto', mt: 3, bgcolor: palette.background.paper, sx: {
            borderRadius: { xs: "0", sm: 2 },
            boxShadow: { xs: "none", sm: 12 },
            width: { xs: "100%", sm: "min(500px, 100vw)" },
        }, children: [_jsx(Box, { width: "min(100px, 25vw)", height: "min(100px, 25vw)", borderRadius: '100%', position: 'absolute', display: 'flex', justifyContent: 'center', alignItems: 'center', left: '50%', top: "-55px", sx: {
                    border: "1px solid black",
                    backgroundColor: profileColors[0],
                    transform: "translateX(-50%)",
                }, children: _jsx(HelpIcon, { fill: profileColors[1], width: '80%', height: '80%' }) }), _jsx(Tooltip, { title: "See all options", children: _jsx(IconButton, { "aria-label": "More", size: "small", onClick: openMoreMenu, sx: {
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                    }, children: _jsx(EllipsisIcon, { fill: palette.background.textSecondary }) }) })] })), [palette.background.paper, palette.background.textSecondary, profileColors, openMoreMenu]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Reminder",
                } }), _jsx(ObjectActionMenu, { actionData: actionData, anchorEl: moreMenuAnchor, object: reminder, onClose: closeMoreMenu, zIndex: zIndex + 1 }), _jsx(Box, { sx: {
                    background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                    display: "flex",
                    paddingTop: 5,
                    paddingBottom: { xs: 0, sm: 2, md: 5 },
                    position: "relative",
                }, children: overviewComponent })] }));
};
//# sourceMappingURL=ReminderView.js.map