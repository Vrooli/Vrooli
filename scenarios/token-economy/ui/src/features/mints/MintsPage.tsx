import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CircleOff, Coins } from "lucide-react";
import { useState } from "react";
import { SupplyPolicy, type TokenType } from "@vrooli/proto-types/token-economy/v1/access/access_pb";

import { minterClient } from "../../api/tokenEconomy";
import { Button } from "@vrooli/react-component-library/Button/2";
import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1";
import { Input } from "../../components/ui/input";
import { Select } from "@vrooli/react-component-library/Select/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ConsoleForm, ConsolePage, Field, RequestState } from "../console/ConsolePage";

const queryKey = ["token-economy", "minter", "token-types"] as const;

export function MintsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [symbol, setSymbol] = useState("");
  const [color, setColor] = useState("");
  const [minterSubject, setMinterSubject] = useState("");
  const [supplyPolicy, setSupplyPolicy] = useState(String(SupplyPolicy.UNBOUNDED));
  const [capAmount, setCapAmount] = useState("0");
  const [mintAmounts, setMintAmounts] = useState<Record<string, string>>({});

  const tokenTypes = useQuery({
    queryKey,
    queryFn: () => minterClient.listTokenTypes({ includeRetired: true }),
  });
  const refresh = () => queryClient.invalidateQueries({ queryKey });
  const createType = useMutation({
    mutationFn: () => minterClient.createTokenType({
      name,
      symbol,
      color,
      minterSubject,
      supplyPolicy: Number(supplyPolicy),
      capAmount: BigInt(capAmount || "0"),
    }),
    onSuccess: async () => {
      setName("");
      setSymbol("");
      await refresh();
    },
  });
  const retireType = useMutation({
    mutationFn: (id: string) => minterClient.retireTokenType({ id }),
    onSuccess: refresh,
  });
  const mintSupply = useMutation({
    mutationFn: ({ id, amount }: { id: string; amount: string }) =>
      minterClient.mintSupply({ tokenTypeId: id, amount: BigInt(amount || "0") }),
    onSuccess: refresh,
  });

  const columns: Array<DataTableColumn<TokenType>> = [
    { id: "identity", header: t(strings.mints.identity), accessor: (row) => <span className="inline-flex items-center gap-2"><span aria-hidden className="h-3 w-3 rounded-full border border-app-border" style={{ backgroundColor: row.color }} />{row.name} <span className="font-mono text-app-muted-foreground">{row.symbol}</span></span>, searchValue: (row) => `${row.name} ${row.symbol}` },
    { id: "supply", header: t(strings.mints.supply), accessor: (row) => <span className="font-mono">{row.mintedAmount.toString()} / {row.capAmount === 0n ? "∞" : row.capAmount.toString()}</span>, sortValue: (row) => Number(row.mintedAmount) },
    { id: "status", header: t(strings.common.status), accessor: (row) => <StatusBadge tone={row.retired ? "neutral" : "success"}>{row.retired ? t(strings.common.retired) : t(strings.common.active)}</StatusBadge> },
    { id: "actions", header: t(strings.common.actions), accessor: (row) => row.retired ? null : <div className="flex flex-wrap gap-2"><Input data-testid={`token-types-mint-amount-${row.id}`} aria-label={t(strings.mints.mintAmount)} className="w-24" min="1" type="number" value={mintAmounts[row.id] ?? "1"} onChange={(event) => setMintAmounts((current) => ({ ...current, [row.id]: event.target.value }))} /><Button data-testid={`token-types-mint-${row.id}`} size="sm" onClick={() => mintSupply.mutate({ id: row.id, amount: mintAmounts[row.id] ?? "1" })}><Coins aria-hidden className="h-4 w-4" />{t(strings.mints.mint)}</Button><Button data-testid={`token-types-retire-${row.id}`} size="sm" variant="danger" onClick={() => retireType.mutate(row.id)}><CircleOff aria-hidden className="h-4 w-4" />{t(strings.common.retire)}</Button></div> },
  ];

  return (
    <ConsolePage testId="page-token-types" title={t(strings.mints.title)} description={t(strings.mints.description)}>
      <ConsoleForm title={t(strings.mints.createTitle)} submitLabel={t(strings.mints.create)} submitTestId="token-types-create" busy={createType.isPending} onSubmit={(event) => { event.preventDefault(); createType.mutate(); }}>
        <Field label={t(strings.common.name)}><Input data-testid="token-types-name" required value={name} onChange={(event) => setName(event.target.value)} /></Field>
        <Field label={t(strings.mints.symbol)}><Input data-testid="token-types-symbol" required value={symbol} onChange={(event) => setSymbol(event.target.value)} /></Field>
        <Field label={t(strings.mints.color)}><Input data-testid="token-types-color" type="color" value={color} onChange={(event) => setColor(event.target.value)} /></Field>
        <Field label={t(strings.mints.minterSubject)}><Input data-testid="token-types-minter-subject" required value={minterSubject} onChange={(event) => setMinterSubject(event.target.value)} /></Field>
        <Field label={t(strings.mints.supplyPolicy)}><Select data-testid="token-types-supply-policy" value={supplyPolicy} onChange={(event) => setSupplyPolicy(event.target.value)} options={[{ value: String(SupplyPolicy.UNBOUNDED), label: t(strings.mints.unbounded) }, { value: String(SupplyPolicy.CAPPED), label: t(strings.mints.capped) }, { value: String(SupplyPolicy.FIXED), label: t(strings.mints.fixed) }]} /></Field>
        <Field label={t(strings.mints.capAmount)}><Input data-testid="token-types-cap" min="0" type="number" value={capAmount} onChange={(event) => setCapAmount(event.target.value)} /></Field>
      </ConsoleForm>
      <RequestState loading={tokenTypes.isLoading} error={tokenTypes.error ?? createType.error ?? retireType.error ?? mintSupply.error} loadingLabel={t(strings.common.loading)} errorLabel={t(strings.common.requestError)} />
      <DataTable rows={tokenTypes.data?.tokenTypes ?? []} columns={columns} getRowKey={(row) => row.id} caption={t(strings.mints.tableCaption)} searchLabel={t(strings.common.search)} searchPlaceholder={t(strings.common.search)} emptyMessage={t(strings.mints.empty)} tableTestId="token-types-list" />
    </ConsolePage>
  );
}
