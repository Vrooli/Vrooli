/** Formats an exact micro-USD amount without converting int64 to imprecise JS number. */
export function formatMicroUSD(amount: bigint) {
  const dollars = amount / 1_000_000n;
  const fraction = (amount % 1_000_000n).toString().padStart(6, "0").replace(/0+$/, "");
  return `$${dollars}${fraction ? `.${fraction}` : ""}`;
}
