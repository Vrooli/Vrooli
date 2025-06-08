import { useTheme } from "@mui/material";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import ListItem from "@mui/material/ListItem";
import ListItemAvatar from "@mui/material/ListItemAvatar";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemText from "@mui/material/ListItemText";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import { alpha } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { useCallback, useMemo } from "react";
import { Icon, type IconInfo } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { type NotificationListItemProps } from "../types.js";

const notificationIconInfo = { name: "NotificationsCustomized", type: "Common" } as const;

// Fallback Icon
const defaultIconInfo: IconInfo = { name: "NotificationsCustomized", type: "Common" };

// Mapping from category to IconInfo using available common icons
const categoryIconMap: Partial<Record<string, IconInfo>> = {
    AccountCreditsOrApi: { name: "Wallet", type: "Common" },
    Award: { name: "Premium", type: "Common" },
    IssueStatus: { name: "Report", type: "Common" },
    Message: { name: "Email", type: "Common" },
    NewObjectInTeam: { name: "Team", type: "Common" },
    NewObjectInProject: { name: "Note", type: "Common" },
    ObjectActivity: { name: "History", type: "Common" },
    Promotion: { name: "Info", type: "Common" },
    PullRequestStatus: defaultIconInfo,
    ReportStatus: { name: "Report", type: "Common" },
    Run: { name: "Play", type: "Common" },
    Schedule: { name: "Today", type: "Common" },
    Security: { name: "Lock", type: "Common" },
    Streak: defaultIconInfo,
    Transfer: { name: "Switch", type: "Common" },
    UserInvite: { name: "Add", type: "Common" },
};

export function NotificationListItem({
    data,
    onAction, // Keep onAction if needed for other actions later, otherwise remove
    loading,
    isSelecting,
    isSelected,
    handleToggleSelect,
    sx, // Pass sx down to the root element
    style, // Pass style down
}: NotificationListItemProps) {
    const { palette, typography } = useTheme();
    const [, setLocation] = useLocation(); // Hook for navigation

    // Determine if the item is read (handle loading state)
    const isRead = !loading && data?.isRead;

    // Click handler for navigation or selection
    const handleClick = useCallback(() => {
        if (loading) return;
        if (isSelecting && handleToggleSelect && data) {
            handleToggleSelect(data);
        } else if (data) {
            // Navigate to notification detail or related object if applicable
            // Example: setLocation(`/notifications/${data.id}`);
            // For now, let's assume no specific navigation on click
            console.log("Notification clicked:", data.id);
        }
    }, [data, isSelecting, handleToggleSelect, loading, setLocation]);

    const handleCheckboxChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        event.stopPropagation(); // Prevent click propagating to the list item
        if (handleToggleSelect && data) {
            handleToggleSelect(data);
        }
    }, [handleToggleSelect, data]);

    // Apply greyed-out style if read
    const itemStyle = {
        opacity: isRead ? 0.6 : 1,
        transition: "opacity 0.3s ease",
        backgroundColor: isSelected ? alpha(palette.secondary.light, 0.3) : undefined,
        "&:hover": {
            backgroundColor: isSelected ? alpha(palette.secondary.light, 0.4) : alpha(palette.action.hover, 0.5),
        },
        // Combine passed sx
        ...sx,
    };

    // Determine title and description
    const title = loading ? <Skeleton variant="text" width="60%" /> : (data?.title ?? "Notification");
    const description = loading ? <Skeleton variant="text" width="90%" /> : (data?.description ?? "");

    // Get the appropriate icon based on category
    const iconInfo = useMemo(() => {
        if (loading || !data?.category) return defaultIconInfo;
        return categoryIconMap[data.category] || defaultIconInfo;
    }, [data?.category, loading]);

    return (
        <ListItem
            disablePadding // Use ListItemButton for padding and hover effects
            sx={itemStyle}
            style={style} // Apply passed style
            secondaryAction={
                isSelecting && !loading ? (
                    // Using a simple Box for click area instead of Checkbox for simplicity
                    <Box
                        onClick={(e) => {
                            e.stopPropagation();
                            if (handleToggleSelect && data) handleToggleSelect(data);
                        }}
                        sx={{ padding: 1, cursor: "pointer" }}
                    >
                        <Box
                            sx={{
                                width: 16,
                                height: 16,
                                borderRadius: "50%",
                                border: `2px solid ${palette.divider}`,
                                backgroundColor: isSelected ? palette.secondary.main : "transparent",
                            }}
                        />
                    </Box>
                ) : null
            }
        >
            <ListItemButton onClick={handleClick} sx={{ paddingY: 1, paddingX: 2 }}>
                <ListItemAvatar sx={{ minWidth: 40, marginRight: 1 }}>
                    {loading ? (
                        <Skeleton variant="circular" width={40} height={40} />
                    ) : (
                        <Avatar sx={{ bgcolor: palette.primary.main }}>
                            <Icon info={iconInfo} fill={palette.primary.contrastText} />
                        </Avatar>
                    )}
                </ListItemAvatar>
                <ListItemText
                    disableTypography
                    primary={
                        <Typography
                            variant="body1"
                            component="div"
                            sx={{ fontWeight: isRead ? typography.fontWeightRegular : typography.fontWeightMedium }}
                        >
                            {title}
                        </Typography>
                    }
                    secondary={
                        <Stack spacing={0.5} sx={{ marginTop: 0.5 }}>
                            {description && <Typography variant="body2" color="text.secondary" sx={{ display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>
                                {description}
                            </Typography>}
                        </Stack>
                    }
                />
            </ListItemButton>
        </ListItem>
    );
}
