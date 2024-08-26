import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkShape, ChatPageTabOption, HistoryPageTabOption, InboxPageTabOption, LINKS, ListObject, MyStuffPageTabOption, SearchPageTabOption, endpointPostBookmark, funcTrue, getObjectUrlBase, noop, shapeBookmark, uuid } from "@local/shared";
import { Box, Button, Divider, IconButton, SwipeableDrawer, Typography, styled, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SiteSearchBar } from "components/inputs/search";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { useFindMany } from "hooks/useFindMany";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSideMenu } from "hooks/useSideMenu";
import { useTabs } from "hooks/useTabs";
import { useWindowSize } from "hooks/useWindowSize";
import { AddIcon, ArrowRightIcon, CloseIcon, CommentIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ArgsType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { LEFT_DRAWER_WIDTH } from "utils/consts";
import { CHAT_SIDE_MENU_ID, PubSub } from "utils/pubsub";
import { ChatTabsInfo, TabParam, chatTabParams } from "utils/search/objectToSearch";
import { ChatCrud } from "views/objects/chat";
import { FindObjectDialog } from "../FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "../types";

export const chatSideMenuDisplayData = {
    persistentOnDesktop: true,
    sideForRightHanded: "left",
} as const;

const SHORT_TAKE = 10;
const emptyArray: readonly [] = [];
const drawerPaperProps = { id: CHAT_SIDE_MENU_ID } as const;
const CHAT_VIEW_ROUTES = [
    LINKS.Chat,
    LINKS.Home,
] as string[];

/**
 * Tab for chat view, which is not available on all pages 
 * (namely ones where the main content is already a chat)
 */
const chatViewTab = {
    color: (palette) => palette.primary.contrastText,
    Icon: CommentIcon,
    key: "Chat",
    titleKey: "Chat",
    where: () => ({}),
} as const;

const NoResultsText = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    fontStyle: "italic",
    padding: theme.spacing(1),
    textAlign: "center",
}));

const TabsBox = styled(Box)(({ theme }) => ({
    background: theme.palette.primary.main,
}));

const SizedDrawer = styled(SwipeableDrawer)(() => ({
    width: LEFT_DRAWER_WIDTH,
    flexShrink: 0,
    "& .MuiDrawer-paper": {
        width: LEFT_DRAWER_WIDTH,
        boxSizing: "border-box",
    },
    "& > .MuiDrawer-root": {
        "& > .MuiPaper-root": {
        },
    },
}));

const searchBarStyle = {
    root: {
        width: "100%",
        "& > .MuiAutocomplete-popper": {
            display: "none",
        },
    },
} as const;

// TODO improve prompts so it searches for actual prompts, rather than just standards. Then update it so when pressed, it adds prompt to chat input
// TODO change tabs based on page type. On DashboardView and ChatCrud, shouldn't have "Chat" tab (since the main page component is already a chat). Otherwise, should have "Chat" tab
// TODO should be disabled on some pages, like SignUpView, SettingsView, etc. On pages where it's not disabled, Navbar should have ListIcon startComponent automatically
export function ChatSideMenu() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [location, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();

    const allTabParams = useMemo(() => {
        const baseTabs = [...chatTabParams];
        if (CHAT_VIEW_ROUTES.includes(location.pathname)) {
            baseTabs.push(chatViewTab as TabParam<ChatTabsInfo>);
        }
        return baseTabs;
    }, [location.pathname]);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "chat-side-tabs", tabParams: allTabParams, display: "dialog" });

    // Handle opening and closing
    const { isOpen, close } = useSideMenu({ id: CHAT_SIDE_MENU_ID, isMobile });
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publish("sideMenu", { id: CHAT_SIDE_MENU_ID, isOpen });
    }, [breakpoints, isOpen]);
    const handleClose = useCallback(() => { close(); }, [close]);

    // Handle adding new bookmarks
    const [isFindBookmarkDialogOpen, setIsFindBookmarkDialogOpen] = useState<boolean>(false);
    const openFindBookmarkDialog = useCallback(() => setIsFindBookmarkDialogOpen(true), []);
    const closeFindBookmarkDialog = useCallback(() => setIsFindBookmarkDialogOpen(false), []);
    const { bookmarkLists } = useMemo(() => getCurrentUser(session), [session]);
    const [addBookmark] = useLazyFetch<BookmarkCreateInput, Bookmark>(endpointPostBookmark);
    const handleBookmarkAdd = useCallback(function handleBookmarkAddCallback(to: BookmarkShape["to"]) {
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

    const addButtonData = useMemo<{ [key in Exclude<ChatPageTabOption, "Chat">]: (() => unknown) }>(() => ({
        History: () => { setLocation(`${getObjectUrlBase({ __typename: "Chat" })}/add`); },
        Favorite: () => { openFindBookmarkDialog(); },
        Prompt: () => { setLocation(`${getObjectUrlBase({ __typename: "Standard" })}/add`); },
        Routine: () => { setLocation(`${getObjectUrlBase({ __typename: "Routine" })}/add`); },
    }), [openFindBookmarkDialog, setLocation]);

    const more1ButtonData = useMemo<{ [key in Exclude<ChatPageTabOption, "Chat">]: (() => unknown) }>(() => ({
        History: () => { setLocation(`${LINKS.Inbox}?type="${InboxPageTabOption.Message}"`); },
        Favorite: () => { setLocation(`${LINKS.History}?type="${HistoryPageTabOption.Bookmarked}"`); },
        Prompt: () => { setLocation(`${LINKS.Search}?type="${SearchPageTabOption.Standard}"`); },
        Routine: () => { setLocation(`${LINKS.Search}?type="${SearchPageTabOption.Routine}"`); },
    }), [setLocation]);

    const more2ButtonData = useMemo<{ [key in Exclude<ChatPageTabOption, "Chat">]: (() => unknown) }>(() => ({
        History: noop,
        Favorite: noop,
        Prompt: () => { setLocation(`${LINKS.MyStuff}?type="${MyStuffPageTabOption.Standard}"`); },
        Routine: () => { setLocation(`${LINKS.MyStuff}?type="${MyStuffPageTabOption.Routine}"`); },
    }), [setLocation]);

    // The "Routine" and "Prompt" tabs have two search results, so we'll have two search hooks
    const { where1, where2 } = useMemo(() => {
        if (typeof where !== "function") {
            return {
                where1: undefined,
                where2: undefined,
            };
        }
        const whereResult = where();
        if (Object.prototype.hasOwnProperty.call(whereResult, "My")) {
            return {
                where1: whereResult.My,
                where2: whereResult.Public,
            };
        }
        return {
            where1: whereResult,
            where2: undefined,
        };
    }, [where]);
    const {
        allData: allData1,
        loading: loading1,
        removeItem: removeItem1,
        searchString,
        setSearchString: setSearchString1,
        updateItem: updateItem1,
    } = useFindMany<ListObject>({
        canSearch: () => currTab.key !== "Chat",
        controlsUrl: false,
        searchType,
        take: SHORT_TAKE,
        where: where1,
    });
    const {
        allData: allData2,
        loading: loading2,
        removeItem: removeItem2,
        setSearchString: setSearchString2,
        updateItem: updateItem2,
    } = useFindMany<ListObject>({
        controlsUrl: false,
        searchType,
        take: SHORT_TAKE,
        where: where2,
    });
    const onAction1 = useCallback((action: keyof ObjectListActions<ListObject>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted":
                removeItem1(...(data as ArgsType<ObjectListActions<ListObject>["Deleted"]>));
                break;
            case "Updated":
                updateItem1(...(data as ArgsType<ObjectListActions<ListObject>["Updated"]>));
                break;
        }
    }, [removeItem1, updateItem1]);
    const onAction2 = useCallback((action: keyof ObjectListActions<ListObject>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted":
                removeItem2(...(data as ArgsType<ObjectListActions<ListObject>["Deleted"]>));
                break;
            case "Updated":
                updateItem2(...(data as ArgsType<ObjectListActions<ListObject>["Updated"]>));
                break;
        }
    }, [removeItem2, updateItem2]);
    const { title1, title2 } = useMemo(() => {
        if (!["Routine", "Prompt"].includes(currTab.key)) {
            return {
                title1: currTab.label,
                title2: undefined,
            };
        }
        return {
            title1: t(`${currTab.key}My`, { count: 2, defaultValue: currTab.label }),
            title2: t(`${currTab.key}Public`, { count: 2, defaultValue: currTab.label }),
        };
    }, [currTab, t]);
    const handleSearchStringChange = useCallback(function handleSearchCallback(newString: string) {
        setSearchString1(newString);
        setSearchString2(newString);
    }, [setSearchString1, setSearchString2]);

    return (
        <>
            <FindObjectDialog
                find="List"
                isOpen={isFindBookmarkDialogOpen}
                handleCancel={closeFindBookmarkDialog}
                handleComplete={handleBookmarkAdd as any}
                limitTo={Object.keys(BookmarkFor) as SelectOrCreateObjectType[]}
            />
            <SizedDrawer
                // Displays opposite of main side menu
                anchor={isLeftHanded ? "right" : "left"}
                open={isOpen}
                onOpen={noop}
                onClose={handleClose}
                PaperProps={drawerPaperProps}
                variant={isMobile ? "temporary" : "persistent"}
            >
                {/* Menu title */}
                <Box
                    display="flex"
                    flexDirection="row"
                    alignItems="center"
                    justifyContent="space-between"
                    height="64px" // Matches Navbar height
                    bgcolor={palette.primary.dark}
                    color={palette.primary.contrastText}
                    p={1}
                >
                    <IconButton
                        aria-label="close"
                        onClick={handleClose}
                    >
                        <CloseIcon fill={palette.primary.contrastText} width="32px" height="32px" />
                    </IconButton>
                    <SiteSearchBar
                        id={"search-bar-chat-side-menu"}
                        isNested={true}
                        placeholder={"Search"}
                        value={searchString}
                        onChange={handleSearchStringChange}
                        onInputChange={noop}
                        sxs={searchBarStyle}
                    />
                </Box>
                <TabsBox>
                    <PageTabs
                        ariaLabel="chat-side-menu-tabs"
                        currTab={currTab}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />
                </TabsBox>
                <Divider />
                <Box overflow="auto" display="flex" flexDirection="column">
                    {currTab.key === "Chat" ? (
                        <ChatCrud display="partial" isCreate={false} />
                    ) : (
                        <>
                            <Box>
                                <Typography variant="h5" p={1}>{title1}</Typography>
                                <Divider />
                                <ObjectList
                                    canNavigate={funcTrue}
                                    dummyItems={new Array(SHORT_TAKE).fill(searchType)}
                                    handleToggleSelect={noop}
                                    hideUpdateButton={true}
                                    isSelecting={false}
                                    items={allData1}
                                    keyPrefix={`chat-search-${currTab.key}-list-item`}
                                    loading={loading1}
                                    onAction={onAction1}
                                    selectedItems={emptyArray}
                                />
                                {allData1.length === 0 && <NoResultsText variant="body1">
                                    {t("NoResults", { ns: "error" })}
                                </NoResultsText>}
                                <Box display="flex" alignItems="center" justifyContent="space-between" pb={4}>
                                    <Button
                                        onClick={addButtonData[currTab.key]}
                                        startIcon={<AddIcon />}
                                        variant="text"
                                    >
                                        {t("Add")}
                                    </Button>
                                    <Button
                                        endIcon={<ArrowRightIcon />}
                                        onClick={more1ButtonData[currTab.key]}
                                        variant="text"
                                    >
                                        {t("More")}
                                    </Button>
                                </Box>
                            </Box>
                            {where2 && <>
                                <Box>
                                    <Typography variant="h5" p={1}>{title2}</Typography>
                                    <Divider />
                                    <ObjectList
                                        canNavigate={funcTrue}
                                        dummyItems={new Array(SHORT_TAKE).fill(searchType)}
                                        handleToggleSelect={noop}
                                        hideUpdateButton={true}
                                        isSelecting={false}
                                        items={allData2}
                                        keyPrefix={`chat-search-${searchType}-list-item`}
                                        loading={loading2}
                                        onAction={onAction2}
                                        selectedItems={emptyArray}
                                    />
                                    {allData2.length === 0 && <NoResultsText variant="body1">
                                        {t("NoResults", { ns: "error" })}
                                    </NoResultsText>}
                                    <Box display="flex" alignItems="center" justifyContent="space-between" pb={4}>
                                        <Button
                                            onClick={addButtonData[currTab.key]}
                                            startIcon={<AddIcon />}
                                            variant="text"
                                        >
                                            {t("Add")}
                                        </Button>
                                        <Button
                                            endIcon={<ArrowRightIcon />}
                                            onClick={more2ButtonData[currTab.key]}
                                            variant="text"
                                        >
                                            {t("More")}
                                        </Button>
                                    </Box>
                                </Box>
                            </>}
                        </>
                    )}
                </Box>
            </SizedDrawer>
        </>
    );
}
