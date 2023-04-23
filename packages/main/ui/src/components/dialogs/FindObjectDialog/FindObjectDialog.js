import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { AddIcon, ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon, UserIcon, VisibleIcon } from "@local/icons";
import { Box, Button, ListItemIcon, ListItemText, Menu, MenuItem, Stack, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { lazily } from "react-lazily";
import { useCustomLazyQuery } from "../../../api";
import { getDisplay } from "../../../utils/display/listTools";
import { getObjectUrl } from "../../../utils/navigation/openObject";
import { addSearchParams, parseSearchParams, removeSearchParams, useLocation } from "../../../utils/route";
import { SearchPageTabOption, SearchType, searchTypeToParams } from "../../../utils/search/objectToSearch";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "../../buttons/SideActionButtons/SideActionButtons";
import { SearchList } from "../../lists/SearchList/SearchList";
import { TIDCard } from "../../lists/TIDCard/TIDCard";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { PageTabs } from "../../PageTabs/PageTabs";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { ShareSiteDialog } from "../ShareSiteDialog/ShareSiteDialog";
const { ApiUpsert } = lazily(() => import("../../../views/api/ApiUpsert/ApiUpsert"));
const { NoteUpsert } = lazily(() => import("../../../views/note/NoteUpsert/NoteUpsert"));
const { OrganizationUpsert } = lazily(() => import("../../../views/organization/OrganizationUpsert/OrganizationUpsert"));
const { ProjectUpsert } = lazily(() => import("../../../views/project/ProjectUpsert/ProjectUpsert"));
const { RoutineUpsert } = lazily(() => import("../../../views/routine/RoutineUpsert/RoutineUpsert"));
const { SmartContractUpsert } = lazily(() => import("../../../views/smartContract/SmartContractUpsert/SmartContractUpsert"));
const { StandardUpsert } = lazily(() => import("../../../views/standard/StandardUpsert/StandardUpsert"));
const tabParams = [{
        Icon: VisibleIcon,
        searchType: "All",
        tabType: "All",
        where: {},
    }, {
        Icon: RoutineIcon,
        searchType: SearchType.Routine,
        tabType: SearchPageTabOption.Routines,
        where: {},
    }, {
        Icon: ProjectIcon,
        searchType: SearchType.Project,
        tabType: SearchPageTabOption.Projects,
        where: {},
    }, {
        Icon: HelpIcon,
        searchType: SearchType.Question,
        tabType: SearchPageTabOption.Questions,
        where: {},
    }, {
        Icon: NoteIcon,
        searchType: SearchType.Note,
        tabType: SearchPageTabOption.Notes,
        where: {},
    }, {
        Icon: OrganizationIcon,
        searchType: SearchType.Organization,
        tabType: SearchPageTabOption.Organizations,
        where: {},
    }, {
        Icon: UserIcon,
        searchType: SearchType.User,
        tabType: SearchPageTabOption.Users,
        where: {},
    }, {
        Icon: StandardIcon,
        searchType: SearchType.Standard,
        tabType: SearchPageTabOption.Standards,
        where: {},
    }, {
        Icon: ApiIcon,
        searchType: SearchType.Api,
        tabType: SearchPageTabOption.Apis,
        where: {},
    }, {
        Icon: SmartContractIcon,
        searchType: SearchType.SmartContract,
        tabType: SearchPageTabOption.SmartContracts,
        where: {},
    }];
const createMap = {
    Api: ApiUpsert,
    Note: NoteUpsert,
    Organization: OrganizationUpsert,
    Project: ProjectUpsert,
    Routine: RoutineUpsert,
    SmartContract: SmartContractUpsert,
    Standard: StandardUpsert,
};
const searchTitleId = "search-vrooli-for-link-title";
export const FindObjectDialog = ({ find, handleCancel, handleComplete, isOpen, limitTo, searchData, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const tabs = useMemo(() => {
        let filteredTabParams = tabParams;
        if (limitTo && limitTo.length > 0) {
            const unversionedLimitTo = limitTo.map(l => l.replace("Version", ""));
            filteredTabParams = tabParams.filter(tab => unversionedLimitTo.includes(tab.searchType));
        }
        return filteredTabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [limitTo, t]);
    const [currTab, setCurrTab] = useState(null);
    useEffect(() => {
        const searchParams = parseSearchParams();
        const index = tabs.findIndex(tab => tab.value === searchParams.type);
        if (index === -1) {
            setCurrTab(tabs[0]);
        }
        else {
            setCurrTab(tabs[index]);
        }
    }, [tabs]);
    console.log("yeeeeet", currTab);
    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        addSearchParams(setLocation, { type: tab.value });
        setCurrTab(tab);
    }, [setLocation]);
    const [createObjectType, setCreateObjectType] = useState(null);
    const [isInviteUserOpen, setIsInviteUserOpen] = useState(false);
    const onInviteUserClose = useCallback(() => setIsInviteUserOpen(false), []);
    const [selectCreateTypeAnchorEl, setSelectCreateTypeAnchorEl] = useState(null);
    const [{ advancedSearchSchema, query }, setSearchParams] = useState({});
    useEffect(() => {
        const fetchParams = async () => {
            const params = searchTypeToParams[createObjectType];
            if (!params)
                return;
            setSearchParams(await params());
        };
        createObjectType !== null && fetchParams();
    }, [createObjectType]);
    const onClose = useCallback((item, versionId) => {
        console.log("onCloseeeeee", item);
        removeSearchParams(setLocation, [
            ...(advancedSearchSchema?.fields.map(f => f.fieldName) ?? []),
            "advanced",
            "sort",
            "time",
        ]);
        if (!item)
            handleCancel();
        else if (find === "Url") {
            const objectUrl = getObjectUrl(item);
            const base = `${window.location.origin}${objectUrl}`;
            const url = versionId ? `${base}/${versionId}` : base;
            if (item) {
                const itemToStore = versionId ? (item.versions?.find(v => v.id === versionId) ?? {}) : item;
                localStorage.setItem(`objectFromUrl:${url}`, JSON.stringify(itemToStore));
            }
            handleComplete((versionId ? `${base}/${versionId}` : base));
        }
        else {
            if (versionId) {
                const version = item.versions?.find(v => v.id === versionId);
                const { versions, ...rest } = item;
                handleComplete({ ...version, root: rest });
            }
            else
                handleComplete(item);
        }
    }, [advancedSearchSchema?.fields, find, handleCancel, handleComplete, setLocation]);
    const [selectedObject, setSelectedObject] = useState(null);
    useEffect(() => {
        setSelectedObject(null);
    }, [isOpen]);
    const { searchType, where } = useMemo(() => {
        console.log("yeet calculating search type", searchData, currTab, tabs);
        if (searchData)
            return searchData;
        if (currTab)
            return { searchType: tabParams.find(tab => tab.tabType === currTab.value)?.searchType ?? "All", where: {} };
        return { searchType: "All", where: {} };
    }, [currTab, searchData, tabs]);
    const onCreateStart = useCallback((e) => {
        e.preventDefault();
        if (searchType === "All" || !currTab)
            setSelectCreateTypeAnchorEl(e.currentTarget);
        else if (searchType === "User")
            setIsInviteUserOpen(true);
        setCreateObjectType(tabParams.find(tab => tab.tabType === currTab.value)?.searchType);
    }, [currTab, searchType]);
    const onSelectCreateTypeClose = useCallback((type) => {
        if (type) {
            if (type === "User") {
                setIsInviteUserOpen(true);
                setSelectCreateTypeAnchorEl(null);
            }
            else {
                setCreateObjectType(type);
            }
        }
        else
            setSelectCreateTypeAnchorEl(null);
    }, []);
    const handleCreated = useCallback((item) => {
        onClose(item);
        setCreateObjectType(null);
    }, [onClose]);
    const handleCreateClose = useCallback(() => {
        setCreateObjectType(null);
    }, []);
    const [getItem, { data: itemData }] = useCustomLazyQuery(query);
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((item, versionId) => {
        if (!query || find !== "Full")
            return false;
        if (itemData && itemData.id === item.id && (!versionId || itemData.versionId === versionId)) {
            onClose(itemData);
        }
        else {
            queryingRef.current = true;
            getItem({ variables: { id: item.id, versionId } });
        }
        return false;
    }, [query, find, itemData, onClose, getItem]);
    const onVersionSelect = useCallback((version) => {
        if (!selectedObject)
            return;
        if (find === "Full") {
            fetchFullData(selectedObject, version.id);
        }
        else {
            onClose(selectedObject, version.id);
        }
    }, [onClose, selectedObject, fetchFullData, find]);
    useEffect(() => {
        if (!query)
            return;
        if (itemData && find === "Full" && queryingRef.current) {
            onClose(itemData);
        }
        queryingRef.current = false;
    }, [onClose, handleCreateClose, itemData, query, find]);
    const onInputSelect = useCallback((newValue) => {
        if (!newValue || newValue.__typename === "Shortcut" || newValue.__typename === "Action")
            return false;
        if (newValue.versions && newValue.versions.length > 0) {
            if (newValue.versions.length === 1) {
                onClose(newValue, newValue.versions[0].id);
                return false;
            }
            setSelectedObject(newValue);
            return false;
        }
        onClose(newValue);
        return false;
    }, [onClose]);
    const CreateView = useMemo(() => ["User", null].includes(createObjectType) ? null : createMap[createObjectType.replace("Version", "")], [createObjectType]);
    useEffect(() => {
        setSelectCreateTypeAnchorEl(null);
    }, [createObjectType]);
    console.log("yeeeet searchType", searchType);
    return (_jsxs(_Fragment, { children: [_jsx(ShareSiteDialog, { onClose: onInviteUserClose, open: isInviteUserOpen, zIndex: zIndex + 2 }), _jsx(LargeDialog, { id: "create-object-dialog", onClose: handleCreateClose, isOpen: createObjectType !== null, titleId: "create-object-dialog-title", zIndex: zIndex + 2, children: CreateView && _jsx(CreateView, { display: "dialog", isCreate: true, onCompleted: handleCreated, onCancel: handleCreateClose, zIndex: zIndex + 2 }) }), !CreateView && _jsx(Menu, { id: "select-create-type-mnu", anchorEl: selectCreateTypeAnchorEl, disableScrollLock: true, open: Boolean(selectCreateTypeAnchorEl), onClose: () => onSelectCreateTypeClose(), children: tabParams.filter((t) => !["All"].includes(t.searchType)).map(tab => (_jsxs(MenuItem, { onClick: () => onSelectCreateTypeClose(tab.searchType), children: [_jsx(ListItemIcon, { children: _jsx(tab.Icon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: t(tab.searchType, { count: 1, defaultValue: tab.searchType }) })] }, tab.searchType))) }), _jsxs(LargeDialog, { id: "resource-find-object-dialog", isOpen: isOpen, onClose: () => { handleCancel(); }, titleId: searchTitleId, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: () => { handleCancel(); }, titleData: {
                            hideOnDesktop: true,
                            titleKey: "SearchVrooli",
                            helpKey: "FindObjectDialogHelp",
                        }, below: tabs.length > 1 && Boolean(currTab) && _jsx(PageTabs, { ariaLabel: "search-tabs", currTab: currTab, onChange: handleTabChange, tabs: tabs }) }), _jsxs(Box, { sx: {
                            minHeight: "500px",
                            margin: { xs: 0, sm: 2 },
                            paddingTop: 4,
                        }, children: [_jsx(SideActionButtons, { display: "dialog", zIndex: zIndex + 1, children: _jsx(ColorIconButton, { "aria-label": "create-new", background: palette.secondary.main, onClick: onCreateStart, children: _jsx(AddIcon, { fill: palette.secondary.contrastText, width: '36px', height: '36px' }) }) }), !selectedObject && _jsx(SearchList, { id: "find-object-search-list", beforeNavigation: onInputSelect, take: 20, resolve: (data) => {
                                    const max = Object.values(data).reduce((acc, val) => {
                                        return Math.max(acc, val.length);
                                    }, -Infinity);
                                    const result = [];
                                    for (let i = 0; i < max; i++) {
                                        for (const key in data) {
                                            if (Array.isArray(data[key]) && data[key][i])
                                                result.push(data[key][i]);
                                        }
                                    }
                                    return result;
                                }, searchType: searchData?.searchType ?? "Popular", zIndex: zIndex, where: searchData?.where ?? { ...where, objectType: searchType === "All" ? undefined : searchType } }), selectedObject && (_jsxs(Stack, { spacing: 2, direction: "column", m: 2, children: [_jsx(Typography, { variant: "h6", mt: 2, textAlign: "center", children: "Select a version" }), [...(selectedObject.versions ?? [])].sort((a, b) => b.versionIndex - a.versionIndex).map((version, index) => (_jsx(TIDCard, { buttonText: t("Select"), description: getDisplay(version).subtitle, Icon: tabParams.find((t) => t.searchType === version.__typename)?.Icon, onClick: () => onVersionSelect(version), title: `${version.versionLabel} - ${getDisplay(version).title}` }, index))), _jsx(Button, { fullWidth: true, color: "secondary", onClick: () => setSelectedObject(null), children: "Select a different object" })] }))] })] })] }));
};
//# sourceMappingURL=FindObjectDialog.js.map