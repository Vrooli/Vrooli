import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { GrantRuleSchema, GrantStatus, RuleCondition, type Grant } from "@vrooli/proto-types/token-economy/v1/access/access_pb";
import { ShieldCheck } from "lucide-react";
import { useState } from "react";

import { minterClient, nextIdempotencyKey } from "../../api/tokenEconomy";
import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1";
import { Input } from "../../components/ui/input";
import { Select } from "@vrooli/react-component-library/Select/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ConsoleForm, ConsolePage, Field, RequestState } from "../console/ConsolePage";

const queryKey = ["token-economy", "minter", "grants"] as const;

export function GrantsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [tokenTypeId, setTokenTypeId] = useState("");
  const [holderId, setHolderId] = useState("");
  const [sourceId, setSourceId] = useState("operator");
  const [authorizer, setAuthorizer] = useState("");
  const [amount, setAmount] = useState("1");
  const [scope, setScope] = useState("");
  const [condition, setCondition] = useState(String(RuleCondition.CATALOG_SCOPE_ALLOWED));
  const [operand, setOperand] = useState("");

  const grants = useQuery({ queryKey, queryFn: () => minterClient.listGrants({ holderId: "", tokenTypeId: "", includeInactive: true }) });
  const createGrant = useMutation({
    mutationFn: () => minterClient.createGrant({
      tokenTypeId,
      holderId,
      grantSourceId: sourceId,
      authorizer,
      amountMinor: BigInt(amount || "0"),
      allowedCatalogScopes: scope ? [scope] : [],
      deniedCatalogScopes: [],
      requiredEvidence: [],
      idempotencyKey: nextIdempotencyKey("grant"),
      rules: [create(GrantRuleSchema, { id: nextIdempotencyKey("rule"), condition: Number(condition), operands: operand ? [operand] : [], amountLimit: BigInt(amount || "0") })],
    }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey }),
  });
  const columns: Array<DataTableColumn<Grant>> = [
    { id: "holder", header: t(strings.grants.holder), accessor: (row) => <code>{row.holderId}</code>, searchValue: (row) => row.holderId },
    { id: "amount", header: t(strings.common.amount), accessor: (row) => <span className="font-mono">{row.amountMinor.toString()}</span>, sortValue: (row) => Number(row.amountMinor) },
    { id: "rules", header: t(strings.grants.rules), accessor: (row) => row.rules.length ? row.rules.map((rule) => <StatusBadge key={rule.id} tone="info"><ShieldCheck aria-hidden className="mr-1 h-3 w-3" />{t(strings.grants.rule)} {rule.condition}</StatusBadge>) : t(strings.grants.noRules) },
    { id: "status", header: t(strings.common.status), accessor: (row) => <StatusBadge tone={row.status === GrantStatus.LIVE ? "success" : "neutral"}>{row.status === GrantStatus.LIVE ? t(strings.common.active) : t(strings.common.inactive)}</StatusBadge> },
  ];
  return (
    <ConsolePage testId="page-grants" title={t(strings.grants.title)} description={t(strings.grants.description)}>
      <ConsoleForm title={t(strings.grants.issueTitle)} submitLabel={t(strings.grants.issue)} submitTestId="grants-issue" busy={createGrant.isPending} onSubmit={(event) => { event.preventDefault(); createGrant.mutate(); }}>
        <Field label={t(strings.grants.tokenType)}><Input data-testid="grants-token-type" required value={tokenTypeId} onChange={(event) => setTokenTypeId(event.target.value)} /></Field>
        <Field label={t(strings.grants.holder)}><Input data-testid="grants-holder" required value={holderId} onChange={(event) => setHolderId(event.target.value)} /></Field>
        <Field label={t(strings.grants.source)}><Input data-testid="grants-source" required value={sourceId} onChange={(event) => setSourceId(event.target.value)} /></Field>
        <Field label={t(strings.grants.authorizer)}><Input data-testid="grants-authorizer" required value={authorizer} onChange={(event) => setAuthorizer(event.target.value)} /></Field>
        <Field label={t(strings.common.amount)}><Input data-testid="grants-amount" min="1" required type="number" value={amount} onChange={(event) => setAmount(event.target.value)} /></Field>
        <Field label={t(strings.grants.catalogScope)}><Input data-testid="grants-scope" value={scope} onChange={(event) => setScope(event.target.value)} /></Field>
        <fieldset data-testid="grants-rule-editor" className="grid gap-3 rounded-control border border-app-border p-3 md:col-span-2"><legend className="px-1 text-sm font-semibold">{t(strings.grants.rules)}</legend><Field label={t(strings.grants.condition)}><Select data-testid="grants-rule-condition" value={condition} onChange={(event) => setCondition(event.target.value)} options={[{ value: String(RuleCondition.CATALOG_SCOPE_ALLOWED), label: t(strings.grants.allowScope) }, { value: String(RuleCondition.CATALOG_SCOPE_DENIED), label: t(strings.grants.denyScope) }, { value: String(RuleCondition.REQUIRED_EVIDENCE), label: t(strings.grants.requireEvidence) }]} /></Field><Field label={t(strings.grants.operand)}><Input data-testid="grants-rule-operand" value={operand} onChange={(event) => setOperand(event.target.value)} /></Field><p data-testid="grants-rule-preview" className="text-sm text-app-muted-foreground">{t(strings.grants.rulePreview, { operand: operand || t(strings.grants.anyScope) })}</p></fieldset>
      </ConsoleForm>
      <RequestState loading={grants.isLoading} error={grants.error ?? createGrant.error} loadingLabel={t(strings.common.loading)} errorLabel={t(strings.common.requestError)} />
      <DataTable rows={grants.data?.grants ?? []} columns={columns} getRowKey={(row) => row.id} caption={t(strings.grants.tableCaption)} searchLabel={t(strings.common.search)} searchPlaceholder={t(strings.common.search)} emptyMessage={t(strings.grants.empty)} tableTestId="grants-table" />
    </ConsolePage>
  );
}
