/* eslint-disable react-perf/jsx-no-new-object-as-prop */
import { FormSchema, RoutineSortBy, RoutineVersionSortBy, TimeFrame } from "@local/shared";
import { useCallback, useState } from "react";
import { SearchButtons } from "./SearchButtons.js";

export default {
    title: "Components/Buttons/SearchButtons",
    component: SearchButtons,
};

const outerStyle = {
    width: "min(400px, 100%)",
    padding: "20px",
    border: "1px solid #ccc",
} as const;
function Outer({ children }: { children: React.ReactNode }) {
    return (
        <div style={outerStyle}>
            {children}
        </div>
    );
}

export function WithAllButtons() {
    const advancedSearchParams = {
        "name": "test",
    };
    const advancedSearchSchema: FormSchema = {
        containers: [],
        elements: [],
    };

    const [sortBy, setSortBy] = useState(RoutineVersionSortBy.CommentsAsc);
    const [timeFrame, setTimeFrame] = useState<TimeFrame | undefined>({ after: new Date(), before: new Date() });
    const onSetSortBy = useCallback((sortBy: string) => {
        setSortBy(sortBy as RoutineVersionSortBy);
    }, [setSortBy]);
    const onSetTimeFrame = useCallback((timeFrame: TimeFrame | undefined) => {
        setTimeFrame(timeFrame);
    }, [setTimeFrame]);

    return (
        <Outer>
            <SearchButtons
                advancedSearchParams={advancedSearchParams}
                advancedSearchSchema={advancedSearchSchema}
                controlsUrl={false}
                searchType="RoutineVersion"
                setAdvancedSearchParams={() => { }}
                setSortBy={onSetSortBy}
                setTimeFrame={onSetTimeFrame}
                sortBy={sortBy}
                sortByOptions={RoutineVersionSortBy}
                timeFrame={timeFrame}
            />
        </Outer>
    );
}
WithAllButtons.parameters = {
    docs: {
        description: {
            story: "Displays all search buttons.",
        },
    },
};


export function WithoutAdvancedSearchParams() {
    return (
        <Outer>
            <SearchButtons
                advancedSearchParams={null}
                advancedSearchSchema={null}
                controlsUrl={false}
                searchType="RoutineVersion"
                setAdvancedSearchParams={() => { }}
                setSortBy={() => { }}
                setTimeFrame={() => { }}
                sortBy={""}
                sortByOptions={RoutineSortBy}
                timeFrame={undefined}
            />
        </Outer>
    );
}
WithoutAdvancedSearchParams.parameters = {
    docs: {
        description: {
            story: "Displays search buttons without advanced search params.",
        },
    },
};
