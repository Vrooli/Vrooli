import { LINKS } from "@local/shared";
import { Box, BoxProps, Divider, List, ListItem, ListItemIcon, ListItemProps, ListItemText, ListProps, styled, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useElementDimensions } from "../../../hooks/useDimensions.js";
import { useLocation } from "../../../route/router.js";
import { pagePaddingBottom } from "../../../styles.js";
import { accountSettingsData, displaySettingsData } from "../../../views/settings/index.js";
import { SettingsData } from "../../../views/settings/types.js";

type ViewSize = "minimal" | "full";

const MINIMIZE_AT_PX = 700;

function isSelected(link: LINKS) {
    return window.location.pathname.includes(link);
}

interface StyledListProps extends ListProps {
    viewSize: ViewSize;
}

const StyledList = styled(List, {
    shouldForwardProp: (prop) => prop !== "viewSize",
})<StyledListProps>(({ theme, viewSize }) => ({
    paddingTop: viewSize === "full" ? theme.spacing(1) : 0,
    paddingBottom: viewSize === "full" ? 1 : 0,
}));

interface StyledListItemProps extends ListItemProps {
    isSelected: boolean;
    viewSize: ViewSize;
}

const StyledListItem = styled(ListItem, {
    shouldForwardProp: (prop) => prop !== "isSelected" && prop !== "viewSize",
})<StyledListItemProps>(({ isSelected, theme, viewSize }) => ({
    transition: "brightness 0.2s ease-in-out",
    background: isSelected ? theme.palette.primary.main : "transparent",
    color: isSelected ? theme.palette.primary.contrastText : theme.palette.background.textPrimary,
    padding: viewSize === "full" ? "4px 12px" : "8px",
    "& >.MuiListItemIcon-root": {
        minWidth: "unset",
    },
}));

interface SettingsListBoxProps extends BoxProps {
    viewSize: ViewSize;
}

type SettingsListItemProps = SettingsData & {
    index: number;
    onSelect: (link: LINKS) => unknown;
    viewSize: ViewSize;
}

const SettingsListBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "viewSize",
})<SettingsListBoxProps>(({ theme, viewSize }) => ({
    width: viewSize === "full" ? "min(100%, 200px)" : "40px",
    height: "fit-content",
    marginLeft: 0,
    marginRight: theme.spacing(1),
    // eslint-disable-next-line no-magic-numbers
    marginTop: theme.spacing(4),
    marginBottom: pagePaddingBottom,
    background: theme.palette.background.paper,
    boxShadow: theme.spacing(2),
    borderRadius: viewSize === "full" ? theme.spacing(2) : theme.spacing(1),
}));

const listItemTextStyle = { marginLeft: 2 } as const;

function SettingsListItem({
    Icon,
    index,
    link,
    onSelect,
    title,
    titleVariables,
    viewSize,
}: SettingsListItemProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const handleSelect = useCallback(function handleSelectCallback() {
        onSelect(link);
    }, [link, onSelect]);

    return (
        <StyledListItem
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            button
            isSelected={isSelected(link)}
            key={index}
            onClick={handleSelect}
            viewSize={viewSize}
        >
            <ListItemIcon>
                <Icon fill={isSelected(link) ? palette.primary.contrastText : palette.background.textSecondary} />
            </ListItemIcon>
            {viewSize === "full" && <ListItemText primary={t(title, { ...titleVariables, defaultValue: title })} sx={listItemTextStyle} />}
        </StyledListItem>
    );
}

export function SettingsList() {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const dimensions = useElementDimensions({ id: "settings-page" });
    const viewSize = useMemo<"minimal" | "full">(() => {
        if (dimensions.width <= MINIMIZE_AT_PX) return "minimal";
        return "full";
    }, [dimensions]);

    const onSelect = useCallback((link: LINKS) => {
        if (!link) return;
        setLocation(link);
    }, [setLocation]);

    const [accountListOpen, setAccountListOpen] = useState(false);
    const toggleAccountList = useCallback(() => setAccountListOpen(!accountListOpen), [accountListOpen]);
    const accountList = useMemo(() => {
        return Object.entries(accountSettingsData).map(([_, data], index) => (
            <SettingsListItem
                {...data}
                index={index}
                key={index}
                onSelect={onSelect}
                viewSize={viewSize}
            />
        ));
    }, [onSelect, viewSize]);

    const [displayListOpen, setDisplayListOpen] = useState(false);
    const toggleDisplayList = useCallback(() => setDisplayListOpen(!displayListOpen), [displayListOpen]);
    const displayList = useMemo(() => {
        return Object.entries(displaySettingsData).map(([_, data], index) => (
            <SettingsListItem
                {...data}
                index={index}
                key={index}
                onSelect={onSelect}
                viewSize={viewSize}
            />
        ));
    }, [onSelect, viewSize]);


    return (
        <SettingsListBox viewSize={viewSize}>
            <StyledList viewSize={viewSize}>
                {viewSize === "full" && <ListItem onClick={toggleAccountList}>
                    <ListItemText primary={t("Account")} />
                </ListItem>}
                <List component="div">
                    {accountList}
                </List>
                <Divider />
                {viewSize === "full" && <ListItem onClick={toggleDisplayList}>
                    <ListItemText primary={t("Display")} />
                </ListItem>}
                <List component="div">
                    {displayList}
                </List>
            </StyledList>
        </SettingsListBox>
    );
}
