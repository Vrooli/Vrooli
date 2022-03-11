import { useQuery } from "@apollo/client";
import { APP_LINKS } from "@local/shared";
import { Box, Typography } from "@mui/material";
import { ResourceListHorizontal } from "components";
import { BaseForm } from "forms";
import { routine } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { useMemo, useState } from "react";
import { getTranslation } from "utils";
import { useLocation, useRoute } from "wouter";
import { SubroutineViewProps } from "../types";

export const SubroutineView = ({
    hasNext,
    hasPrevious,
    partialData,
    session,
}: SubroutineViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Run}/:routineId/:subroutineId`);
    const { routineId, subroutineId } = useMemo(() => ({
        routineId: params?.routineId ?? "",
        subroutineId: params?.subroutineId ?? "",
    }), [params]);
    // Fetch data
    const { data, loading } = useQuery<routine>(routineQuery, { variables: { input: { id: routineId } } });

    const { description, instructions, name } = useMemo(() => {
        const languages = navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true) ?? getTranslation(partialData, 'description', languages, true),
            instructions: getTranslation(data, 'instructions', languages, true) ?? getTranslation(partialData, 'instructions', languages, true),
            name: getTranslation(data, 'name', languages, true) ?? getTranslation(partialData, 'name', languages, true),
        }
    }, [data, partialData]);

    // The schema for the form
    const [schema, setSchema] = useState<any>();

    return (
        <Box sx={{ background: (t) => t.palette.background.paper}}>
            {/* Resources */}
            {data?.routine?.resourceLists && <ResourceListHorizontal 
                list={data.routine.resourceLists[0]}
                canEdit={false}
                session={session}
            /> }
            {/* Description */}
            {description && (<Typography variant="body1">{description}</Typography>)}
            {/* Form generate from routine's inputs list */}
            <BaseForm
                schema={schema}
                onSubmit={(values) => console.log(values)}
            />
        </Box>
    )
}