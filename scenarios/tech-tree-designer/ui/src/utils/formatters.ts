export const formatCurrency = (value?: number | null): string => {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '$0'
  }
  if (value >= 1e12) return `$${(value / 1e12).toFixed(1)}T`
  if (value >= 1e9) return `$${(value / 1e9).toFixed(1)}B`
  if (value >= 1e6) return `$${(value / 1e6).toFixed(1)}M`
  return `$${value.toLocaleString()}`
}

export const formatTimestamp = (value?: string | null): string => {
  if (!value) {
    return 'Not synced yet'
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return 'Timestamp unavailable'
  }
  return parsed.toLocaleString()
}
