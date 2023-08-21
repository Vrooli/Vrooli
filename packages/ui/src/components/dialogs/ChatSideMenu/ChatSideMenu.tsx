import { CommonKey } from "@local/shared";
import { Box, IconButton, Palette, SwipeableDrawer, useTheme } from "@mui/material";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SessionContext } from "contexts/SessionContext";
import { useFindMany } from "hooks/useFindMany";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useSideMenu } from "hooks/useSideMenu";
import { useTabs } from "hooks/useTabs";
import { useWindowSize } from "hooks/useWindowSize";
import { CloseIcon, CommentIcon, RoutineIcon, SearchIcon, StandardIcon } from "icons";
import React, { useCallback, useContext, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { ListObject } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { ChatPageTabOption, SearchType } from "utils/search/objectToSearch";

type ChatSideMenuType = "Chat" | "Routine" | "Prompt";
type ChatSideMenuObject = Chat | Routine | Standard;

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
    const session = useContext(SessionContext);
    const { t } = useTranslation();
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

    const {
        allData,
        loading,
        loadMore,
        setAllData,
    } = useFindMany<ChatSideMenuObject>({
        searchType,
        where: where(),
    });

    // Handle opening and closing
    const { isOpen, close } = useSideMenu(id, isMobile);
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publishSideMenu({ id, isOpen });
    }, [breakpoints, isOpen]);

    const handleClose = useCallback((event: React.MouseEvent<HTMLElement>) => {
        close();
    }, [close]);

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
                    background: palette.background.default,
                    overflowY: "auto",
                    borderRight: palette.mode === "light" ? "none" : `1px solid ${palette.divider}`,
                },
                "& > .MuiDialog-container": {
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
                    width: "min(400px, 100%)",
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
                    onClick={() => { }}
                >
                    <SearchIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                </IconButton>
            </Box>
            <ListContainer
                emptyText=""
                isEmpty={allData.length === 0 && !loading}
            >
                <ObjectList
                    dummyItems={new Array(10).fill(searchType)}
                    items={allData as ListObject[]}
                    keyPrefix={`${searchType}-list-item`}
                    loading={loading}
                // onAction={onAction}
                // onClick={(item) => onClick(item)}
                />
            </ListContainer>
        </SwipeableDrawer>
    );
};
