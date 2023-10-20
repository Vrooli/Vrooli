import { Box, IconButton, SwipeableDrawer, useTheme } from "@mui/material";
import { SelectorBase } from "components/inputs/SelectorBase/SelectorBase";
import { SearchList } from "components/lists/SearchList/SearchList";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useSideMenu } from "hooks/useSideMenu";
import { useTabs } from "hooks/useTabs";
import { useWindowSize } from "hooks/useWindowSize";
import { CloseIcon, SearchIcon } from "icons";
import React, { useCallback, useEffect, useState } from "react";
import { noSelect } from "styles";
import { noop } from "utils/objects";
import { PubSub } from "utils/pubsub";
import { ChatPageTabOption, chatTabParams } from "utils/search/objectToSearch";

export const chatSideMenuDisplayData = {
    persistentOnDesktop: true,
    sideForRightHanded: "left",
} as const;

const zIndex = 1300;
const id = "chat-side-menu";

export const ChatSideMenu = () => {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<ChatPageTabOption>({ id: "chat-side-tabs", tabParams: chatTabParams, display: "dialog" });

    // Handle opening and closing
    const { isOpen, close } = useSideMenu(id, isMobile);
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publishSideMenu({ id, isOpen });
    }, [breakpoints, isOpen]);
    const handleClose = useCallback((event: React.MouseEvent<HTMLElement>) => {
        close();
    }, [close]);

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);
    // If showing search filter, focus the search input
    useEffect(() => {
        if (!showSearchFilters) return;
        const searchInput = document.getElementById("search-bar-chat-related-search-list");
        searchInput?.focus();
    }, [showSearchFilters]);


    return (
        <SwipeableDrawer
            // Displays opposite of main side menu
            anchor={isLeftHanded ? "right" : "left"}
            open={isOpen}
            onOpen={noop}
            onClose={handleClose}
            PaperProps={{ id }}
            variant={isMobile ? "temporary" : "persistent"}
            sx={{
                zIndex,
                "& .MuiDrawer-paper": {
                    background: palette.background.default,
                    overflowY: "auto",
                    borderRight: palette.mode === "light" ? "none" : `1px solid ${palette.divider}`,
                    width: "min(350px, 100%)",
                },
                "& > .MuiDialog-container": {
                    width: "min(350px, 100%)",
                    "& > .MuiPaper-root": {
                        zIndex,
                    },
                },
            }}
        >
            {/* Menu title with close icon, list type selector, and search icon */}
            <Box
                sx={{
                    ...noSelect,
                    display: "flex",
                    flexDirection: "row",
                    alignItems: "center",
                    justifyContent: "space-between",
                    padding: 1,
                    gap: 1,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    textAlign: "center",
                    fontSize: { xs: "1.5rem", sm: "2rem" },
                    height: "64px", // Matches Navbar height
                    paddingRight: 3, // Matches navbar padding
                }}
            >
                <IconButton
                    aria-label="close"
                    onClick={handleClose}
                >
                    <CloseIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                </IconButton>
                <SelectorBase
                    name="tab"
                    value={currTab}
                    label=""
                    onChange={(tab) => handleTabChange(undefined, tab)}
                    options={tabs}
                    getOptionLabel={(o) => o.label}
                    fullWidth={true}
                />
                <IconButton
                    aria-label="search"
                    onClick={toggleSearchFilters}
                >
                    <SearchIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                </IconButton>
            </Box>
            <Box sx={{ overflowY: "auto" }} >
                <SearchList
                    id="chat-related-search-list"
                    display={isMobile ? "dialog" : "partial"}
                    dummyLength={10}
                    hideUpdateButton={true}
                    take={20}
                    searchType={searchType}
                    sxs={{
                        ...(showSearchFilters ?
                            { search: { marginTop: 2 } } :
                            {
                                search: { display: "none" },
                                buttons: { display: "none" },
                            }
                        ),
                        listContainer: {
                            borderRadius: 0,
                            // Hide Avatar from list items
                            "& .MuiAvatar-root": { display: "none" },
                        },
                    }}
                    where={where()}
                />
            </Box>
        </SwipeableDrawer>
    );
};
