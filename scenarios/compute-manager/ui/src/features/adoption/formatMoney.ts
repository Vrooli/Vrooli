/** Format ISO 4217 minor units using the currency's own fraction precision. */
export function formatMinorMoney(minor: number, currency: string, locale: string | undefined, minorUnitsLabel: string): string {
  try {
    const code = currency.toUpperCase();
    const supportedValuesOf = (Intl as typeof Intl & {
      supportedValuesOf?: (key: "currency") => string[];
    }).supportedValuesOf;
    if (!supportedValuesOf?.("currency").includes(code)) {
      return `${code} ${minor} ${minorUnitsLabel}`;
    }
    const formatter = new Intl.NumberFormat(locale, { style: "currency", currency: code });
    const digits = formatter.resolvedOptions().maximumFractionDigits;
    if (digits == null) return `${code} ${minor} ${minorUnitsLabel}`;
    return formatter.format(minor / (10 ** digits));
  } catch {
    return `${currency.toUpperCase()} ${minor} ${minorUnitsLabel}`;
  }
}
