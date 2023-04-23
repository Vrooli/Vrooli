import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { AddIcon, EditIcon } from "@local/icons";
import { Box, TextField, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { bookmarkListFindOne } from "../../../api/generated/endpoints/bookmarkList_findOne";
import { ColorIconButton } from "../../../components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "../../../components/buttons/SideActionButtons/SideActionButtons";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { ObjectAction } from "../../../utils/actions/objectActions";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { useLocation } from "../../../utils/route";
export const BookmarkListView = ({ display = "page", partialData, zIndex = 200, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { object: existing, setObject: setBookmarkList } = useObjectFromUrl({
        query: bookmarkListFindOne,
        partialData,
    });
    const { label } = useMemo(() => ({ label: existing?.label ?? "" }), [existing]);
    useEffect(() => {
        document.title = `${label} | Vrooli`;
    }, [label]);
    const actionData = useObjectActions({
        object: existing,
        objectType: "BookmarkList",
        setLocation,
        setObject: setBookmarkList,
    });
    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((event) => {
        setSearchString(event.target.value);
    }, []);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "BookmarkList",
                    titleVariables: { count: 1 },
                }, below: _jsx(TextField, { placeholder: "Search bookmarks...", autoFocus: true, value: searchString, onChange: updateSearchString }) }), _jsxs(_Fragment, { children: [_jsxs(SideActionButtons, { display: display, zIndex: zIndex + 1, children: [_jsx(ColorIconButton, { "aria-label": "Edit list", background: palette.secondary.main, onClick: () => { actionData.onActionStart(ObjectAction.Edit); }, children: _jsx(EditIcon, { fill: palette.secondary.contrastText, width: '36px', height: '36px' }) }), _jsx(ColorIconButton, { "aria-label": "Add bookmark", background: palette.secondary.main, onClick: () => { }, children: _jsx(AddIcon, { fill: palette.secondary.contrastText, width: '36px', height: '36px' }) })] }), _jsx(Box, { sx: {
                            display: "flex",
                            justifyContent: "center",
                            alignItems: "center",
                        } })] })] }));
};
//# sourceMappingURL=BookmarkListView.js.map