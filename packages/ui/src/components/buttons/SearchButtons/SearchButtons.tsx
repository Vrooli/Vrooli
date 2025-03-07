import { Box } from "@mui/material";
import { useMemo } from "react";
import { AdvancedSearchButton } from "../AdvancedSearchButton/AdvancedSearchButton.js";
import { SortButton } from "../SortButton/SortButton.js";
import { TimeButton } from "../TimeButton/TimeButton.js";
import { SearchButtonsProps } from "../types.js";

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
    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            marginBottom: 1,
            ...sx,
        } as const;
    }, [sx]);

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
