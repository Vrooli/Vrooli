function decimal(value: string): { numerator: bigint; denominator: bigint } | undefined {
  const normalized = value.trim();
  if (!/^\d+(\.\d+)?$/.test(normalized)) return undefined;
  const [whole, fraction = ""] = normalized.split(".");
  const denominator = 10n ** BigInt(fraction.length);
  return { numerator: BigInt(whole + fraction), denominator };
}

export function scaleAmount(amount: string, canonicalYield: string, targetYield: string): string | undefined {
  const source = decimal(amount);
  const canonical = decimal(canonicalYield);
  const target = decimal(targetYield);
  if (!source || !canonical || !target || canonical.numerator === 0n) return undefined;
  const numerator = source.numerator * target.numerator * canonical.denominator;
  const denominator = source.denominator * target.denominator * canonical.numerator;
  const whole = numerator / denominator;
  let remainder = numerator % denominator;
  if (remainder === 0n) return whole.toString();
  let fraction = "";
  for (let index = 0; index < 6 && remainder !== 0n; index += 1) {
    remainder *= 10n;
    fraction += (remainder / denominator).toString();
    remainder %= denominator;
  }
  fraction = fraction.replace(/0+$/, "");
  return fraction ? `${whole.toString()}.${fraction}` : whole.toString();
}
