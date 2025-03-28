import { FormBuilder, FormInputBase, FormSchema, ParseSearchParamsResult, SearchType, TimeFrame, TranslationFuncCommon, TranslationKeyCommon, parseSearchParams } from "@local/shared";
import { Box, Button, Grid, Menu, MenuItem, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { Formik } from "formik";
import i18next from "i18next";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormRunView } from "../../../forms/FormView/FormView.js";
import { usePopover } from "../../../hooks/usePopover.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { addSearchParams, removeSearchParams } from "../../../route/searchParams.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { convertFormikForSearch, convertSearchForFormik } from "../../../utils/search/inputToSearch.js";
import { searchTypeToParams } from "../../../utils/search/objectToSearch.js";
import { LargeDialog } from "../../dialogs/LargeDialog/LargeDialog.js";
import { DateRangeMenu } from "../../lists/DateRangeMenu/DateRangeMenu.js";
import { TopBar } from "../../navigation/TopBar.js";
import { BottomActionsGrid } from "../BottomActionsGrid.js";
import { SearchButtonsProps } from "../types.js";

export const StyledSearchButton = styled(Box)<{ active?: boolean }>(
    ({ theme, active }) => ({
        display: "flex",
        alignItems: "center",
        borderRadius: "24px",
        cursor: "pointer",
        padding: "2px 12px",
        margin: "2px",
        "&:hover": {
            filter: "brightness(1.05)",
        },
        // Conditionally apply styles based on `active`
        ...(active
            ? {
                // When there is a sortBy
                backgroundColor: theme.palette.secondary.main,
                border: "none",
                color: theme.palette.secondary.contrastText,
                boxShadow: theme.shadows[1],
            }
            : {
                // When there is NO sortBy
                backgroundColor: "transparent",
                border: `1px solid ${theme.palette.secondary.main}`,
                color: theme.palette.secondary.main,
            }),
    }),
);

export type LabelledSortOption<SortBy> = { label: string, value: SortBy };

const sortButtonId = "sort-results-button";
const sortMenuListId = "sort-results-menu-list";
const sortMenuListProps = {
    "aria-label": "Sort search results",
    "aria-labelledby": sortMenuListId,
    "aria-describedby": sortButtonId,
} as const;

interface SortMenuProps {
    anchorEl: Element | null;
    onClose: (label?: string, value?: string) => unknown;
    sortOptions: LabelledSortOption<string>[];
}

function SortMenu({
    anchorEl,
    onClose,
    sortOptions,
}: SortMenuProps) {
    const { t } = useTranslation();
    const open = Boolean(anchorEl);

    function handleClose() {
        onClose();
    }

    const menuItems = useMemo(() => {
        const menuItems: JSX.Element[] = [];
        sortOptions.forEach(option => {
            const optionLabel = t(`${option.value}` as TranslationKeyCommon);

            function handleClick() {
                onClose(optionLabel, option.value);
            }

            if (optionLabel) {
                menuItems.push(
                    <MenuItem
                        key={option.value}
                        value={option.value}
                        onClick={handleClick}
                    >
                        {optionLabel}
                    </MenuItem>,
                );
            }
        });
        return menuItems;
    }, [sortOptions, t, onClose]);

    return (
        <Menu
            id={sortMenuListId}
            disableScrollLock={true}
            anchorEl={anchorEl}
            open={open}
            onClose={handleClose}
            MenuListProps={sortMenuListProps}
        >
            {menuItems}
        </Menu>
    );
}

interface SortButtonProps {
    options: any; // No way to specify generic enum
    setSortBy: (sortBy: string) => unknown;
    sortBy: string;
}
export function SortButton({
    options,
    setSortBy,
    sortBy,
}: SortButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [sortAnchorEl, openSort, closeSort] = usePopover();
    function handleSortClose(_label?: string, selected?: string) {
        closeSort();
        if (selected) setSortBy(selected);
    }

    /** Wrap options with labels */
    const sortOptionsLabelled = useMemo<LabelledSortOption<string>[]>(() => {
        if (!options) return [];
        return Object.keys(options).map((key) => ({
            label: (i18next.t(key as TranslationKeyCommon, key)) as unknown as string,
            value: key,
        }));
    }, [options]);

    /** Find sort by label when sortBy changes */
    const sortByLabel = useMemo(() => t(sortBy as TranslationKeyCommon, sortBy), [sortBy, t]);
    const isActive = Boolean(sortBy);

    return (
        <>
            <SortMenu
                sortOptions={sortOptionsLabelled}
                anchorEl={sortAnchorEl}
                onClose={handleSortClose}
            />
            <Tooltip title={t("SortBy")} placement="top">
                <StyledSearchButton
                    aria-label={t("SortBy")}
                    id={sortButtonId}
                    active={isActive}
                    onClick={openSort}
                >
                    <IconCommon
                        decorative
                        fill={
                            isActive
                                ? palette.secondary.contrastText
                                : palette.secondary.main
                        }
                        name="Sort"
                    />
                    <Typography variant="body2" ml={0.5}>{sortByLabel}</Typography>
                </StyledSearchButton>
            </Tooltip>
        </>
    );
}

/** Map time selections to time length in milliseconds */
const timeOptions = {
    "TimeAll": undefined,
    "TimeYear": 31536000000,
    "TimeMonth": 2592000000,
    "TimeWeek": 604800000,
    "TimeDay": 86400000,
    "TimeHour": 3600000,
} as const;
const timeMenuListId = "results-time-menu-list";
const timeButtonId = "time-filter-button";
const timeMenuListProps = {
    "aria-label": "Filters search results by time created or updated",
    "aria-labelledby": timeMenuListId,
    "aria-describedby": timeButtonId,
} as const;

interface TimeMenuProps {
    anchorEl: Element | null;
    onClose: (labelKey?: TranslationKeyCommon, timeFrame?: { after?: Date, before?: Date }) => unknown;
}

function TimeMenu({
    anchorEl,
    onClose,
}: TimeMenuProps) {
    const { t } = useTranslation();

    const open = Boolean(anchorEl);
    function handleClose() {
        onClose();
    }

    const [customRangeAnchorEl, openCustomRange, closeCustomRange] = usePopover();

    const menuItems = useMemo(() => Object.keys(timeOptions).map((labelKey) => {
        function handleClick() {
            // If All is selected, pass undefined to onClose
            if (!timeOptions[labelKey]) onClose(labelKey as TranslationKeyCommon);
            // Otherwise, pass the time as object with "after"
            else onClose(labelKey as TranslationKeyCommon, { after: new Date(Date.now() - timeOptions[labelKey]) });
        }

        return (
            <MenuItem
                key={labelKey}
                value={timeOptions[labelKey]}
                onClick={handleClick}
            >
                {t(labelKey as TranslationKeyCommon)}
            </MenuItem>
        );
    }), [onClose, t]);

    function handleDateRangeSubmit(after: Date | undefined, before: Date | undefined) {
        onClose("Custom", { after, before });
    }

    return (
        <Menu
            id={timeMenuListId}
            anchorEl={anchorEl}
            disableScrollLock={true}
            open={open}
            onClose={handleClose}
            MenuListProps={timeMenuListProps}
        >
            {menuItems}
            <MenuItem
                id='custom-range-menu-item'
                value='custom'
                onClick={openCustomRange}
            >
                {t("CustomRange")}
            </MenuItem>
            <DateRangeMenu
                anchorEl={customRangeAnchorEl}
                onClose={closeCustomRange}
                onSubmit={handleDateRangeSubmit}
            />
        </Menu>
    );
}

interface TimeButtonProps {
    setTimeFrame: (timeFrame: TimeFrame | undefined) => unknown;
    timeFrame: TimeFrame | undefined;
}
export function TimeButton({
    setTimeFrame,
    timeFrame,
}: TimeButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [timeFrameLabel, setTimeFrameLabel] = useState<string>("");

    const [timeAnchorEl, openTime, closeTime] = usePopover();
    function handleTimeClose(labelKey?: TranslationKeyCommon, frame?: TimeFrame) {
        closeTime();
        setTimeFrame(frame);
        if (labelKey) setTimeFrameLabel(t(labelKey));
    }

    const isActive = Boolean(timeFrame);

    return (
        <>
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
            />
            <Tooltip title={t("TimeCreated")} placement="top">
                <StyledSearchButton
                    aria-label={t("TimeCreated")}
                    id={timeButtonId}
                    onClick={openTime}
                    active={Boolean(timeFrame)}
                >
                    <IconCommon
                        decorative
                        fill={
                            isActive
                                ? palette.secondary.contrastText
                                : palette.secondary.main
                        }
                        name="History"
                    />
                    <Typography variant="body2" ml={0.5}>{timeFrameLabel}</Typography>
                </StyledSearchButton>
            </Tooltip>
        </>
    );
}

function createTopBarOptions(resetForm: (() => unknown), t: TranslationFuncCommon) {
    return [
        {
            iconInfo: { name: "Refresh", type: "Common" } as const,
            label: t("Reset"),
            onClick: resetForm,
        },
    ];
}

const FormContainer = styled(Box)(({ theme }) => ({
    margin: "auto",
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    padding: theme.spacing(2),
    paddingBottom: "64px",
}));

function AdvancedSearchDialog({
    handleClose,
    handleSearch,
    isOpen,
    searchType,
}: {
    handleClose: () => unknown;
    handleSearch: (searchQuery: ParseSearchParamsResult) => unknown;
    isOpen: boolean;
    searchType: SearchType | `${SearchType}`;
}) {
    const { t } = useTranslation();

    const [searchParams, setSearchParams] = useState<ParseSearchParamsResult>(parseSearchParams());
    // Search schema to use
    const [schema, setSchema] = useState<FormSchema | null>(null);
    useEffect(() => {
        setSchema(searchType in searchTypeToParams ? searchTypeToParams[searchType]().advancedSearchSchema : null);
    }, [searchType]);

    const initialValues = useMemo(function initialValuesMemo() {
        // Calculate initial values from schema, to use for values not in URL
        const initialValues = FormBuilder.generateInitialValues(schema?.elements);
        // Parse search params from URL
        const urlValues = convertSearchForFormik(searchParams, schema);
        // Replace default values with URL values
        for (const key in urlValues) {
            if (urlValues[key] === undefined) continue;
            initialValues[key] = urlValues[key] as never;
        }
        return initialValues;
    }, [schema, searchParams]);
    const validationSchema = useMemo(function validationSchemaMemo() {
        return schema ? FormBuilder.generateYupSchema(schema) : undefined;
    }, [schema]);

    const onSubmit = useCallback(function onSubmitCallback(values: ParseSearchParamsResult) {
        if (schema) {
            const searchValue = convertFormikForSearch(values, schema);
            handleSearch(searchValue);
        }
        handleClose();
        setSearchParams(parseSearchParams());
    }, [handleSearch, schema, handleClose]);

    return (
        <LargeDialog
            id={ELEMENT_IDS.AdvancedSearchDialog}
            isOpen={isOpen}
            onClose={handleClose}
            titleId={ELEMENT_IDS.AdvancedSearchDialogTitle}
        >
            <Formik<ParseSearchParamsResult>
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={onSubmit}
                validationSchema={validationSchema}
            >
                {(formik) => {
                    function onSubmit() {
                        formik.handleSubmit();
                    }
                    function resetForm() {
                        formik.setValues(FormBuilder.generateInitialValues(schema?.elements));
                    }
                    const topBarOptions = createTopBarOptions(resetForm, t);

                    return (
                        <>
                            <TopBar
                                display="dialog"
                                onClose={handleClose}
                                tabTitle={t("AdvancedSearch")}
                                titleId={ELEMENT_IDS.AdvancedSearchDialogTitle}
                                options={topBarOptions}
                            />
                            <FormContainer>
                                {/* Search options */}
                                {schema && <FormRunView
                                    disabled={false}
                                    schema={schema}
                                />}
                            </FormContainer>
                            {/* Search/Cancel buttons */}
                            <BottomActionsGrid display="dialog">
                                <Grid item xs={6}>
                                    <Button
                                        fullWidth
                                        startIcon={<IconCommon
                                            decorative
                                            name="Search"
                                        />}
                                        type="submit"
                                        onClick={onSubmit}
                                        variant="contained"
                                    >{t("Search")}</Button>
                                </Grid>
                                <Grid item xs={6}>
                                    <Button
                                        fullWidth
                                        startIcon={<IconCommon
                                            decorative
                                            name="Cancel"
                                        />}
                                        onClick={handleClose}
                                        variant="outlined"
                                    >{t("Cancel")}</Button>
                                </Grid>
                            </BottomActionsGrid>
                        </>
                    );
                }}
            </Formik>
        </LargeDialog>
    );
}

const filterCountLabelStyle = { marginLeft: 0.5 } as const;

interface AdvancedSearchButtonProps {
    advancedSearchParams: object | null;
    advancedSearchSchema: FormSchema | null | undefined;
    controlsUrl: boolean;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (params: object | null) => unknown;
}
export function AdvancedSearchButton({
    advancedSearchParams,
    advancedSearchSchema,
    controlsUrl,
    searchType,
    setAdvancedSearchParams,
}: AdvancedSearchButtonProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Update params when schema changes
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (!advancedSearchSchema?.elements) {
            setAdvancedSearchParams(null);
            return;
        }
        // Open advanced search dialog, if needed
        if (typeof searchParams.advanced === "boolean") setAdvancedSearchDialogOpen(searchParams.advanced);
        // Any search params that aren't advanced, search, sort, or time MIGHT be advanced search params
        const { _advanced, _search, _sort, _time, ...otherParams } = searchParams;
        // Find valid advanced search params
        const allAdvancedSearchParams = advancedSearchSchema.elements.filter(f => Object.prototype.hasOwnProperty.call(f, "fieldName")).map(f => (f as FormInputBase).fieldName);
        // fields in both otherParams and allAdvancedSearchParams should be the new advanced search params
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [advancedSearchSchema?.elements, setAdvancedSearchParams]);

    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true); }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        setAdvancedSearchDialogOpen(false);
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: ParseSearchParamsResult) => {
        if (!controlsUrl) return;
        // Remove schema fields from search params
        removeSearchParams(setLocation, advancedSearchSchema?.elements?.filter(f => Object.prototype.hasOwnProperty.call(f, "fieldName")).map(f => (f as FormInputBase).fieldName) ?? []);
        // Add set fields to search params
        addSearchParams(setLocation, values);
        setAdvancedSearchParams(values);
    }, [advancedSearchSchema?.elements, controlsUrl, setAdvancedSearchParams, setLocation]);

    // Set dialog open stats in url search params
    useEffect(() => {
        if (!controlsUrl) return;
        addSearchParams(setLocation, { advanced: advancedSearchDialogOpen });
    }, [advancedSearchDialogOpen, controlsUrl, setLocation]);

    const isActive = typeof advancedSearchParams === "object" && advancedSearchParams !== null && Object.keys(advancedSearchParams).length > 0;

    return (
        <>
            <AdvancedSearchDialog
                handleClose={handleAdvancedSearchDialogClose}
                handleSearch={handleAdvancedSearchDialogSubmit}
                isOpen={advancedSearchDialogOpen}
                searchType={searchType}
            />
            {advancedSearchParams && <Tooltip title={t("SeeAllSearchSettings")} placement="top">
                <StyledSearchButton
                    onClick={handleAdvancedSearchDialogOpen}
                    active={isActive}
                >
                    <IconCommon
                        decorative
                        fill={
                            isActive
                                ? palette.secondary.contrastText
                                : palette.secondary.main
                        }
                        name="Build"
                    />
                    {Object.keys(advancedSearchParams).length > 0 && <Typography variant="body2" sx={filterCountLabelStyle}>
                        *{Object.keys(advancedSearchParams).length}
                    </Typography>}
                </StyledSearchButton>
            </Tooltip>}
        </>
    );
}

export function SearchButtons({
    advancedSearchParams,
    advancedSearchSchema,
    controlsUrl,
    searchType,
    setAdvancedSearchParams,
    setSortBy,
    setTimeFrame,
    sortBy,
    sortByOptions,
    sx,
    timeFrame,
}: SearchButtonsProps) {
    const { spacing } = useTheme();

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            gap: spacing(1),
            ...sx,
        } as const;
    }, [spacing, sx]);

    return (
        <Box sx={outerBoxStyle}>
            <SortButton
                options={sortByOptions}
                setSortBy={setSortBy}
                sortBy={sortBy}
            />
            <TimeButton
                setTimeFrame={setTimeFrame}
                timeFrame={timeFrame}
            />
            {searchType !== "Popular" && <AdvancedSearchButton
                advancedSearchParams={advancedSearchParams}
                advancedSearchSchema={advancedSearchSchema}
                controlsUrl={controlsUrl}
                searchType={searchType}
                setAdvancedSearchParams={setAdvancedSearchParams}
            />}
        </Box>
    );
}
