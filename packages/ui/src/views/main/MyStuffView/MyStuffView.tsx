/**
 * Search page for organizations, projects, routines, standards, and users
 */
import { GqlModelType } from "@local/shared";
import { IconButton, ListItemIcon, ListItemText, Menu, MenuItem, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SessionContext } from "contexts/SessionContext";
import { useTabs } from "hooks/useTabs";
import { AddIcon, SearchIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { scrollIntoFocusedView } from "utils/display/scroll";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { MyStuffPageTabOption, myStuffTabParams, SearchType } from "utils/search/objectToSearch";
import { MyStuffViewProps } from "../types";

export const MyStuffView = ({
    isOpen,
    onClose,
}: MyStuffViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const {
        id: userId,
        apisCount,
        membershipsCount,
        questionsAskedCount,
        smartContractsCount,
        standardsCount,
    } = useMemo(() => getCurrentUser(session), [session]);
    const display = toDisplay(isOpen);

    /**
     * Filter out certain tabs that we don't have any data for, 
     * so user isn't overwhelmed with tabs for objects they never worked with. 
     * Always keeps routines, projects, and notes
     */
    const filteredTabs = useMemo(() => myStuffTabParams.filter(tab => {
        switch (tab.tabType) {
            case MyStuffPageTabOption.Api:
                return Boolean(apisCount);
            case MyStuffPageTabOption.Organization:
                return Boolean(membershipsCount);
            case MyStuffPageTabOption.Question:
                return Boolean(questionsAskedCount);
            case MyStuffPageTabOption.SmartContract:
                return Boolean(smartContractsCount);
            case MyStuffPageTabOption.Standard:
                return Boolean(standardsCount);
        }
        return true;
    }), [apisCount, membershipsCount, questionsAskedCount, smartContractsCount, standardsCount]);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<MyStuffPageTabOption>({ tabParams: filteredTabs, display });

    // Menu for selection object type to create
    const [selectCreateTypeAnchorEl, setSelectCreateTypeAnchorEl] = useState<null | HTMLElement>(null);

    const onCreateStart = useCallback((e: React.MouseEvent<HTMLElement>) => {
        // If tab is 'All', open menu to select type
        if (searchType === SearchType.Popular) {
            setSelectCreateTypeAnchorEl(e.currentTarget);
            return;
        }
        // Navigate to object's add page
        setLocation(`${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`);
    }, [searchType, setLocation]);
    const onSelectCreateTypeClose = useCallback((type?: SearchType) => {
        if (type) setLocation(`${getObjectUrlBase({ __typename: type as `${GqlModelType}` })}/add`);
        else setSelectCreateTypeAnchorEl(null);
    }, [setLocation]);

    const focusSearch = () => { scrollIntoFocusedView("search-bar-my-stuff-list"); };

    return (
        <>
            <Menu
                id="select-create-type-menu"
                anchorEl={selectCreateTypeAnchorEl}
                disableScrollLock={true}
                open={Boolean(selectCreateTypeAnchorEl)}
                onClose={() => onSelectCreateTypeClose()}
            >
                {/* Never show 'All' */}
                {myStuffTabParams.filter((t) => ![SearchType.Popular].includes(t.searchType)).map(tab => (
                    <MenuItem
                        key={tab.searchType}
                        onClick={() => onSelectCreateTypeClose(tab.searchType as SearchType)}
                    >
                        <ListItemIcon>
                            <tab.Icon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={t(tab.searchType, { count: 1, defaultValue: tab.searchType })} />
                    </MenuItem>
                ))}
            </Menu>
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={t("MyStuff")}
                below={<PageTabs
                    ariaLabel="my-stuff-tabs"
                    fullWidth
                    id="my-stuff-tabs"
                    ignoreIcons
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                id="my-stuff-list"
                display={display}
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                where={where({ userId: userId ?? "" })}
                sxs={{ search: { marginTop: 2 } }}
            />}
            <SideActionsButtons
                display={display}
                sx={{ position: "fixed" }}
            >
                <IconButton aria-label={t("FilterList")} onClick={focusSearch} sx={{ background: palette.secondary.main }}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
                {userId ? (
                    <IconButton aria-label={t("Add")} onClick={onCreateStart} sx={{ background: palette.secondary.main }}>
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
};
