import { jsx as _jsx } from "react/jsx-runtime";
import { Box } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { useLocation } from "../../../utils/route";
import { translateSearchItems } from "../../../utils/search/siteToSearch";
import { SessionContext } from "../../../utils/SessionContext";
import { SettingsSearchBar } from "../../inputs/search";
import { TopBar } from "../TopBar/TopBar";
const searchItems = [
    {
        label: "Profile",
        keywords: ["Bio", "Handle", "Name"],
        value: "Profile",
    },
    {
        label: "Privacy",
        keywords: ["History"],
        value: "Privacy",
    },
    {
        label: "Authentication",
        keywords: [{ key: "Wallet", count: 1 }, { key: "Wallet", count: 2 }, { key: "Email", count: 1 }, { key: "Email", count: 2 }, "LogOut", "Security"],
        value: "authentication",
    },
    {
        label: "Display",
        keywords: ["Theme", "Light", "Dark", "Interests", "Hidden", { key: "Tag", count: 1 }, { key: "Tag", count: 2 }, "History"],
        value: "Display",
    },
    {
        label: "Notification",
        labelArgs: { count: 2 },
        keywords: [{ key: "Alert", count: 1 }, { key: "Alert", count: 2 }, { key: "PushNotification", count: 1 }, { key: "PushNotification", count: 2 }],
        value: "Notification",
    },
    {
        label: "Schedule",
        keywords: [],
        value: "Schedule",
    },
];
export const SettingsTopBar = ({ display, onClose, titleData, }) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const [searchString, setSearchString] = useState("");
    const updateSearch = useCallback((newValue) => { setSearchString(newValue); }, []);
    const onInputSelect = useCallback((newValue) => {
        if (!newValue)
            return;
        setLocation(newValue);
    }, [setLocation]);
    const searchOptions = useMemo(() => translateSearchItems(searchItems, session), [session]);
    return (_jsx(TopBar, { display: display, onClose: onClose, titleData: titleData, below: _jsx(Box, { sx: {
                width: "min(100%, 700px)",
                margin: "auto",
                marginTop: 2,
                marginBottom: 2,
                paddingLeft: 2,
                paddingRight: 2,
            }, children: _jsx(SettingsSearchBar, { value: searchString, onChange: updateSearch, onInputChange: onInputSelect, options: searchOptions }) }) }));
};
//# sourceMappingURL=SettingsTopBar.js.map