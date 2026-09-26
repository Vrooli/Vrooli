import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchVulnerabilities } from "../lib/api";

export const useVulnerabilities = (credentialData?: { resource_statuses?: Array<{ resource_name: string }> }) => {
  const [componentType, setComponentType] = useState("");
  const [componentFilter, setComponentFilter] = useState("");
  const [severityFilter, setSeverityFilter] = useState("");

  const vulnerabilityQuery = useQuery({
    queryKey: ["vulnerabilities", componentType, componentFilter, severityFilter],
    queryFn: () =>
      fetchVulnerabilities({
        componentType: componentType || undefined,
        component: componentFilter || undefined,
        severity: severityFilter || undefined
      }),
    refetchInterval: 90000
  });

  const componentOptions = useMemo(() => {
    const set = new Set<string>();
    credentialData?.resource_statuses?.forEach((status) => set.add(status.resource_name));
    vulnerabilityQuery.data?.vulnerabilities?.forEach((vuln) => set.add(vuln.component_name));
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [credentialData, vulnerabilityQuery.data]);

  return {
    vulnerabilityQuery,
    componentType,
    componentFilter,
    severityFilter,
    componentOptions,
    setComponentType,
    setComponentFilter,
    setSeverityFilter
  };
};
