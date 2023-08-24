import { User } from "@local/shared";
import { Box, Stack, Typography, useTheme } from "@mui/material";
import { ProfileGroupProps } from "components/types";
import { SessionContext } from "contexts/SessionContext";
import { BotIcon, UserIcon } from "icons";
import { useContext, useMemo } from "react";
import { SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session";

type IconType = SvgComponent | string | null;

const maxIconsDisplayed = 4;

const renderIcon = (Icon: IconType, key: number) => {
    if (Icon === null) return <Box sx={{ width: 24, height: 24 }} key={key} />;
    if (typeof Icon === "string") return <Typography variant="body2" key={key} sx={{ width: 24, height: 24 }}>{Icon}</Typography>;
    return <Icon key={key} fill="white" width={24} height={24} />;
};

/**
 * Displays up to 4 profiles in a square. Useful for list items of 
 * objects that have a list of users, such as chats, meetings, and roles
 */
export const ProfileGroup = ({
    sx,
    users = [],
}: ProfileGroupProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();

    const icons = useMemo<[IconType, IconType, IconType, IconType]>(() => {
        let newIcons: IconType[] = [];
        let maxUserIcons = maxIconsDisplayed;
        // If there are more members than allowed, add a "+X" icon
        const hasMoreMembers = users.length > maxUserIcons;
        if (hasMoreMembers) maxUserIcons--;
        // Filter out yourself, and add the first X members
        newIcons = users
            .filter((user: User) => user.id !== getCurrentUser(session)?.id)
            .slice(0, maxUserIcons)
            .map((user: User) => user.isBot ? BotIcon : UserIcon); // TODO Replace with User's profile pic in the future
        // Add the "+X" icon if there are more members than allowed
        if (hasMoreMembers) newIcons.push(`+${users.length - maxUserIcons}`);
        // Add null icons to fill the remaining space up to the max
        while (newIcons.length < maxIconsDisplayed) newIcons.push(null);
        return newIcons as [IconType, IconType, IconType, IconType];
    }, [session, users]);

    return (
        <Box
            sx={{
                background: palette.primary.light,
                width: { xs: "58px", md: "69px" },
                height: { xs: "58px", md: "69px" },
                overflow: "hidden",
                borderRadius: "12px",
                color: "white",
                position: "relative",
            }}
        >
            {/* Members & add members icons */}
            <Stack direction="column" justifyContent="center" alignItems="center" style={{ height: "100%", width: "100%" }}>
                <Stack direction="row" justifyContent="space-around">
                    {renderIcon(icons[0], 0)}
                    {renderIcon(icons[1], 1)}
                </Stack>
                <Stack direction="row" justifyContent="space-around">
                    {renderIcon(icons[2], 2)}
                    {renderIcon(icons[3], 3)}
                </Stack>
            </Stack>
        </Box>
    );
};
