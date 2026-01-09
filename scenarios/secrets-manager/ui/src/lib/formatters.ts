export const formatTimestamp = (value?: string) => {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};

export const percentage = (found: number, total: number) => {
  if (!total) return 0;
  return Math.round((found / total) * 100);
};
