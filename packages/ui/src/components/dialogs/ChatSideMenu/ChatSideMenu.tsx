import { Bookmark, BookmarkCreateInput, BookmarkFor, CommonKey, endpointPostBookmark, noop, uuid } from "@local/shared";
import { Box, IconButton, SwipeableDrawer, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { SearchList } from "components/lists/SearchList/SearchList";
import { SessionContext } from "contexts/SessionContext";
import { useFindMany } from "hooks/useFindMany";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSideMenu } from "hooks/useSideMenu";
import { useTabs } from "hooks/useTabs";
import { useWindowSize } from "hooks/useWindowSize";
import { useZIndex } from "hooks/useZIndex";
import { AddIcon, CloseIcon, SearchIcon } from "icons";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { noSelect } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { ListObject } from "utils/display/listTools";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { ChatPageTabOption, chatTabParams } from "utils/search/objectToSearch";
import { BookmarkShape, shapeBookmark } from "utils/shape/models/bookmark";
import { FindObjectDialog } from "../FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "../types";

export const chatSideMenuDisplayData = {
    persistentOnDesktop: true,
    sideForRightHanded: "left",
} as const;

const id = "chat-side-menu";

export const ChatSideMenu = ({
    idPrefix,
}: {
    /** Alters menu ID so that the menu has its own pub/sub events */
    idPrefix?: string
}) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: `${idPrefix ?? ""}chat-side-tabs`, tabParams: chatTabParams, display: "dialog" });

    // Handle opening and closing
    const { isOpen, close } = useSideMenu({ id, idPrefix, isMobile });
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publish("sideMenu", { id, idPrefix, isOpen });
    }, [breakpoints, idPrefix, isOpen]);
    const handleClose = useCallback((event: React.MouseEvent<HTMLElement>) => {
        close();
    }, [close]);

    const [zIndex, handleTransitionExit] = useZIndex(isOpen, true, 1000);

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);
    // If showing search filter, focus the search input
    useEffect(() => {
        if (!showSearchFilters) return;
        const searchInput = document.getElementById("search-bar-chat-related-search-list");
        searchInput?.focus();
    }, [showSearchFilters]);

    // Handle adding new bookmarks
    const [isFindBookmarkDialogOpen, setIsFindBookmarkDialogOpen] = useState<boolean>(false);
    const openFindBookmarkDialog = useCallback(() => setIsFindBookmarkDialogOpen(true), []);
    const closeFindBookmarkDialog = useCallback(() => setIsFindBookmarkDialogOpen(false), []);
    const { bookmarkLists } = useMemo(() => getCurrentUser(session), [session]);
    const [addBookmark] = useLazyFetch<BookmarkCreateInput, Bookmark>(endpointPostBookmark);
    const handleBookmarkAdd = useCallback((to: BookmarkShape["to"]) => {
        let bookmarkListId: string | undefined;
        if (bookmarkLists && bookmarkLists.length) {
            // Try to find "Favorites" bookmark list first
            const favorites = bookmarkLists.find(list => list.label === "Favorites");
            if (favorites) {
                bookmarkListId = favorites.id;
            } else {
                // Otherwise, just use the first bookmark list
                bookmarkListId = bookmarkLists[0].id;
            }
        }
        fetchLazyWrapper<BookmarkCreateInput, Bookmark>({
            fetch: addBookmark,
            inputs: shapeBookmark.create({
                __typename: "Bookmark" as const,
                id: uuid(),
                to,
                list: {
                    __typename: "BookmarkList",
                    id: bookmarkListId ?? uuid(),
                    // Setting label marks this as a create, 
                    // which should only be done if there is no bookmarkListId
                    label: bookmarkListId ? undefined : "Favorites",
                },
            }),
            onSuccess: () => {
                //TODO add to list
            },
        });
    }, [addBookmark, bookmarkLists]);

    const tabToAddData: { [key in ChatPageTabOption]?: readonly [CommonKey, (() => unknown)] } = {
        Chat: ["NewChat", () => { setLocation(`${getObjectUrlBase({ __typename: "Chat" })}/add`); }],
        Favorite: ["AddBookmark", () => { openFindBookmarkDialog(); }],
        PromptMy: ["CreatePrompt", () => { setLocation(`${getObjectUrlBase({ __typename: "Standard" })}/add`); }],
        RoutineMy: ["CreateRoutine", () => { setLocation(`${getObjectUrlBase({ __typename: "Routine" })}/add`); }],
    } as const;

    const findManyData = useFindMany<ListObject>({
        controlsUrl: false,
        searchType,
        take: 20,
        where: where(),
    });

    return (
        <>
            <FindObjectDialog
                find="List"
                isOpen={isFindBookmarkDialogOpen}
                handleCancel={closeFindBookmarkDialog}
                handleComplete={handleBookmarkAdd as any}
                limitTo={Object.keys(BookmarkFor) as SelectOrCreateObjectType[]}
            />
            <SwipeableDrawer
                // Displays opposite of main side menu
                anchor={isLeftHanded ? "right" : "left"}
                open={isOpen}
                onOpen={noop}
                onClose={handleClose}
                onTransitionExited={handleTransitionExit}
                PaperProps={{ id }}
                variant={isMobile ? "temporary" : "persistent"}
                sx={{
                    zIndex,
                    "& .MuiDrawer-paper": {
                        background: palette.background.paper,
                        overflowY: "auto",
                        borderRight: palette.mode === "light" ? "none" : `1px solid ${palette.divider}`,
                        width: "min(250px, 100%)",
                        zIndex,
                    },
                    "& > .MuiDrawer-root": {
                        width: "min(250px, 100%)",
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
                        padding: 1,
                        gap: 1,
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                        textAlign: "center",
                        fontSize: { xs: "1.5rem", sm: "2rem" },
                        height: "64px", // Matches Navbar height
                    }}
                >
                    <IconButton
                        aria-label="close"
                        onClick={handleClose}
                        sx={{ marginRight: "auto" }}

                    >
                        <CloseIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                    </IconButton>
                    <Tooltip title={t("SearchFiltersShow")}>
                        <IconButton
                            aria-label="search"
                            onClick={toggleSearchFilters}
                        >
                            <SearchIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                        </IconButton>
                    </Tooltip>
                    {tabToAddData[currTab.key] && <Tooltip title={t(tabToAddData[currTab.key]![0])}>
                        <IconButton aria-label="add" onClick={tabToAddData[currTab.key]![1]}>
                            <AddIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                        </IconButton>
                    </Tooltip>}
                </Box>
                <Box sx={{ overflowY: "auto" }} >
                    <SelectorBase
                        color={palette.background.textPrimary}
                        name="tab"
                        value={currTab}
                        label=""
                        onChange={(tab) => handleTabChange(undefined, tab)}
                        options={tabs}
                        getOptionLabel={(o) => o.label}
                        fullWidth={true}
                    />
                    <SearchList
                        {...findManyData}
                        id="chat-related-search-list"
                        display={isMobile ? "dialog" : "partial"}
                        dummyLength={10}
                        hideUpdateButton={true}
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
                            },
                        }}
                    />
                </Box>
            </SwipeableDrawer>
        </>
    );
};
