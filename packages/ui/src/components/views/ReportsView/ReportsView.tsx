import { ReportsViewProps } from "../types";
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useLazyQuery, useQuery } from "@apollo/client";
import { reports, reportsVariables } from "graphql/generated/reports";
import { useState, useEffect } from "react";
import { reportsQuery } from "graphql/query";
import { validate as uuidValidate } from 'uuid';

import { Report } from "types";

export const ReportsView = ({
    session
}: ReportsViewProps) => {
    const [, params] = useRoute(`${APP_LINKS.Routine}/reports/:id`);
    const id = params?.id;

    const { loading, error, data } = useQuery<reports, reportsVariables>(
        reportsQuery,
        {
            variables: {
                input: { routineId: id },
            },
        },
    );

    if (loading) {
        return <p>Loading reports...</p>
    }
    if (error) {
        console.error(error);
        return <p>Error</p>;
    }

    if (!data || !data.reports) {
        return <></>
    }

    return data.reports.edges.map((edge, i) => {
        const report = edge.node;
        return <div key={i}>
            <p>{report.reason}</p>
            <p>{report.details}</p>
        </div>
    });
}