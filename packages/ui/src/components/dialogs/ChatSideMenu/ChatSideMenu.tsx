import { IconButton, List, SwipeableDrawer, useTheme } from "@mui/material";
import { Stack } from "@mui/system";
import { SessionContext } from "contexts/SessionContext";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useWindowSize } from "hooks/useWindowSize";
import { CloseIcon } from "icons";
import React, { useCallback, useContext, useEffect, useState } from "react";
import { noSelect } from "styles";
import { PubSub } from "utils/pubsub";

export const ChatSideMenu = () => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();

    // Handle opening and closing
    const [isOpen, setIsOpen] = useState(false);
    useEffect(() => {
        const sideMenuSub = PubSub.get().subscribeChatSideMenu((open) => {
            setIsOpen(o => open ?? !o);
        });
        return (() => {
            PubSub.get().unsubscribe(sideMenuSub);
        });
    }, []);
    const close = useCallback(() => { setIsOpen(false); }, []);

    const handleClose = useCallback((event: React.MouseEvent<HTMLElement>) => {
        close();
    }, [close]);

    return (
        <SwipeableDrawer
            // Displays opposite of main side menu
            anchor={(isMobile && isLeftHanded) ? "right" : "left"}
            open={isOpen}
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            onOpen={() => { }}
            onClose={handleClose}
            sx={{
                zIndex: 20000,
                "& .MuiDrawer-paper": {
                    background: palette.background.default,
                    overflowY: "auto",
                },
            }}
        >
            {/* Menu title with close icon */}
            <Stack
                direction='row'
                spacing={1}
                sx={{
                    ...noSelect,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    padding: 1,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    textAlign: "center",
                    fontSize: { xs: "1.5rem", sm: "2rem" },
                    height: "64px", // Matches Navbar height
                    paddingRight: 3, // Matches navbar padding
                }}
            >
                {/* Close icon */}
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={handleClose}
                    sx={{
                        marginLeft: (isMobile && isLeftHanded) ? "auto" : "unset",
                        marginRight: (isMobile && isLeftHanded) ? "unset" : "auto",
                    }}
                >
                    <CloseIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                </IconButton>
            </Stack>
            {/* Chat */}
            {/* Icons to switch between chats, prompts (standards), and routines */}
            {/* TODO */}
            {/* List of other chats with the current user, prompts you can use, or routines you can run with the users (if they're bots) */}
            <List id="chat-side-menu-list" sx={{ paddingTop: 0, paddingBottom: 0 }}>
                {/* TODO */}
            </List>
        </SwipeableDrawer>
    );
};
