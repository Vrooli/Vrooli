import { Box } from "@mui/material"
import { AdvancedSearchButton, SortButton, TimeButton } from "components/buttons"
import { SearchButtonsListProps } from "../types"

export const SearchButtonsList = ({
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
}: SearchButtonsListProps) => {
    return (
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', marginBottom: 1 }}>
            <SortButton
                options={sortByOptions}
                setSortBy={setSortBy}
                session={session}
                sortBy={sortBy}
            />
            <TimeButton
                setTimeFrame={setTimeFrame}
                session={session}
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