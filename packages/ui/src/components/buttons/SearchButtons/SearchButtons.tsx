import { Box } from "@mui/material"
import { AdvancedSearchButton, SortButton, TimeButton } from "components/buttons"
import { SearchButtonsProps } from "../types"

export const SearchButtons = ({
    advancedSearchParams,
    advancedSearchSchema,
    searchType,
    session,
    setAdvancedSearchParams,
    setSortBy,
    setTimeFrame,
    sortBy,
    sortByOptions,
    timeFrame,
    zIndex,
}: SearchButtonsProps) => {
    return (
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', marginBottom: 1 }}>
            <SortButton
                options={sortByOptions}
                setSortBy={setSortBy}
                sortBy={sortBy}
            />
            <TimeButton
                setTimeFrame={setTimeFrame}
                timeFrame={timeFrame}
            />
            <AdvancedSearchButton
                advancedSearchParams={advancedSearchParams}
                advancedSearchSchema={advancedSearchSchema}
                searchType={searchType}
                setAdvancedSearchParams={setAdvancedSearchParams}
                session={session}
                zIndex={zIndex}
            />
        </Box>
    )
}