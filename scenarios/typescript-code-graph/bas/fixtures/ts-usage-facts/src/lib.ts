export default function formatLabel(value: string): string {
  return value.toUpperCase();
}

export function useFeatureFlag(name: string): boolean {
  return name.length > 0;
}

export const palette = {
  primary: "blue",
};

export type ReactNode = string;
