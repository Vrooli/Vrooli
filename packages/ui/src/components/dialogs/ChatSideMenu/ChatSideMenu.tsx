import { CommonKey } from "@local/shared";
import { Box, IconButton, Palette, SwipeableDrawer, useTheme } from "@mui/material";
import { SearchList } from "components/lists/SearchList/SearchList";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useSideMenu } from "hooks/useSideMenu";
import { useTabs } from "hooks/useTabs";
import { useWindowSize } from "hooks/useWindowSize";
import { CloseIcon, CommentIcon, RoutineIcon, SearchIcon, StandardIcon } from "icons";
import React, { useCallback, useEffect, useState } from "react";
import { noSelect } from "styles";
import { PubSub } from "utils/pubsub";
import { ChatPageTabOption, SearchType } from "utils/search/objectToSearch";

export const chatSideMenuDisplayData = {
    persistentOnDesktop: true,
    sideForRightHanded: "left",
} as const;

const tabParams = [{
    Icon: CommentIcon,
    color: (palette: Palette) => palette.primary.contrastText,
    titleKey: "Chat" as CommonKey,
    searchType: SearchType.Chat,
    tabType: ChatPageTabOption.Chat,
    where: () => ({}),
}, {
    Icon: RoutineIcon,
    color: (palette: Palette) => palette.primary.contrastText,
    titleKey: "Routine" as CommonKey,
    searchType: SearchType.Routine,
    tabType: ChatPageTabOption.Routine,
    where: () => ({}),
}, {
    Icon: StandardIcon,
    color: (palette: Palette) => palette.primary.contrastText,
    titleKey: "Prompt" as CommonKey,
    searchType: SearchType.Standard,
    tabType: ChatPageTabOption.Prompt,
    where: () => ({}),
}];

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
    } = useTabs<ChatPageTabOption>({ tabParams, display: "dialog" });

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
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            onOpen={() => { }}
            onClose={handleClose}
            PaperProps={{ id }}
            variant={isMobile ? "temporary" : "persistent"}
            sx={{
                zIndex,
                "& .MuiDrawer-paper": {
                    width: "min(350px, 100%)",
                    background: palette.background.default,
                    overflowY: "auto",
                    borderRight: palette.mode === "light" ? "none" : `1px solid ${palette.divider}`,
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
                <Box sx={{
                    // border: `1px solid ${palette.primary.contrastText}`,
                    background: palette.primary.light,
                    borderRadius: 2,
                    overflow: "overlay",
                    margin: 1,
                }}>
                    <PageTabs
                        ariaLabel="chat-related-tabs"
                        fullWidth
                        id="chat-related-tabs"
                        currTab={currTab}
                        onChange={handleTabChange}
                        tabs={tabs}
                        sx={{ height: "40px" }}
                    />
                </Box>
                <IconButton
                    aria-label="search"
                    onClick={toggleSearchFilters}
                >
                    <SearchIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                </IconButton>
            </Box>
            <Box sx={{ overflowY: "auto" }} >
                {searchType && <SearchList
                    id="chat-related-search-list"
                    display="partial"
                    dummyLength={10}
                    take={20}
                    searchType={searchType}
                    sxs={showSearchFilters ? {
                        search: { marginTop: 2 },
                        listContainer: { borderRadius: 0 },
                    } : {
                        search: { display: "none" },
                        buttons: { display: "none" },
                        listContainer: { borderRadius: 0 },
                    }}
                    where={where()}
                />}
            </Box>
        </SwipeableDrawer>
    );
};
