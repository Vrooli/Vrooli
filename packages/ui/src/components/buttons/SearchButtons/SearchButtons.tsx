import { SearchType } from "@local/shared";
import { Box } from "@mui/material";
import { AdvancedSearchButton } from "../AdvancedSearchButton/AdvancedSearchButton";
import { SortButton } from "../SortButton/SortButton";
import { TimeButton } from "../TimeButton/TimeButton";
import { SearchButtonsProps } from "../types";

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
    return (
        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", marginBottom: 1, ...sx }}>
            <SortButton
                options={sortByOptions}
                setSortBy={setSortBy}
                sortBy={sortBy}
            />
            <TimeButton
                setTimeFrame={setTimeFrame}
                timeFrame={timeFrame}
            />
            {searchType !== SearchType.Popular && <AdvancedSearchButton
                advancedSearchParams={advancedSearchParams}
                advancedSearchSchema={advancedSearchSchema}
                controlsUrl={controlsUrl}
                searchType={searchType}
                setAdvancedSearchParams={setAdvancedSearchParams}
            />}
        </Box>
    );
}
