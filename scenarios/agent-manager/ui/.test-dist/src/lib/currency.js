const usdFormatters = new Map();
function getUsdFormatter(options) {
    const key = [
        options?.minimumFractionDigits ?? "",
        options?.maximumFractionDigits ?? "",
        options?.useGrouping ?? "",
    ].join("|");
    const cached = usdFormatters.get(key);
    if (cached)
        return cached;
    const formatter = new Intl.NumberFormat("en-US", {
        style: "currency",
        currency: "USD",
        minimumFractionDigits: options?.minimumFractionDigits,
        maximumFractionDigits: options?.maximumFractionDigits,
        useGrouping: options?.useGrouping,
    });
    usdFormatters.set(key, formatter);
    return formatter;
}
export function formatUsd(value, options) {
    return getUsdFormatter(options).format(value);
}
export function formatUsdFixed(value, fractionDigits, options) {
    return formatUsd(value, {
        minimumFractionDigits: fractionDigits,
        maximumFractionDigits: fractionDigits,
        useGrouping: options?.useGrouping,
    });
}
export function formatPricingUsdPerMillion(price) {
    if (price === undefined || price === 0)
        return "-";
    if (price < 0.01) {
        return formatUsdFixed(price, 4, { useGrouping: false });
    }
    return formatUsdFixed(price, 2, { useGrouping: false });
}
