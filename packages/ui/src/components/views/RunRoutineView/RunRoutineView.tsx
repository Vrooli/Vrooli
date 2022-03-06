import { useQuery } from "@apollo/client";
import { APP_LINKS } from "@local/shared";
import { Typography } from "@mui/material";
import { ResourceListHorizontal } from "components";
import { BaseForm } from "forms";
import { routine } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { useMemo, useState } from "react";
import { useLocation, useRoute } from "wouter";
import { RunRoutineViewProps } from "../types";

export const RunRoutineView = ({
    hasNext,
    hasPrevious,
    partialData,
}: RunRoutineViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Orchestrate}/run/:id`);
    const [, params2] = useRoute(`${APP_LINKS.Run}/:id`);
    const id: string = useMemo(() => params?.id ?? params2?.id ?? '', [params, params2]);
    // Fetch data
    const { data, loading } = useQuery<routine>(routineQuery, { variables: { input: { id } } });

    // The schema for the form
    const [schema, setSchema] = useState<any>();

    return (
        <>
            {/* Resources */}
            {data?.routine?.resources && <ResourceListHorizontal /> }
            {/* Description */}
            {data?.routine?.description && (<Typography variant="body1">{data?.routine?.description}</Typography>)}
            {/* Form generate from routine's inputs list */}
            <BaseForm
                schema={schema}
                onSubmit={(values) => console.log(values)}
            />
        </>
    )
}