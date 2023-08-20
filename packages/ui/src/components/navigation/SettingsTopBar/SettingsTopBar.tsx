import { Box } from "@mui/material";
import { SettingsSearchBar } from "components/inputs/search";
import { SessionContext } from "contexts/SessionContext";
import { useCallback, useContext, useMemo, useState } from "react";
import { useLocation } from "route";
import { PreSearchItem, translateSearchItems } from "utils/search/siteToSearch";
import { TopBar } from "../TopBar/TopBar";
import { SettingsTopBarProps } from "../types";

/**
 * Search bar options
 */
const searchItems: PreSearchItem[] = [
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

/**
 * Generates an app bar for both pages and dialogs
 */
export const SettingsTopBar = ({
    display,
    onClose,
    ...rest
}: SettingsTopBarProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const [searchString, setSearchString] = useState<string>("");

    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue); }, []);
    const onInputSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        setLocation(newValue);
    }, [setLocation]);

    const searchOptions = useMemo(() => translateSearchItems(searchItems, session), [session]);

    return (
        <TopBar
            {...rest}
            display={display}
            hideTitleOnDesktop={true}
            onClose={onClose}
            // Search bar to find settings
            below={<Box sx={{
                width: "min(100%, 700px)",
                margin: "auto",
                marginTop: 2,
                marginBottom: 2,
                paddingLeft: 2,
                paddingRight: 2,
            }}>
                <SettingsSearchBar
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    options={searchOptions}
                />
            </Box>}
        />
    );
};
