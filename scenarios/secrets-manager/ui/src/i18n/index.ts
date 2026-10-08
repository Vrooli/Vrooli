import en from "./locales/en.json";

const catalogue: Record<string, string> = {};
for (const [namespace, values] of Object.entries(en as Record<string, unknown>)) {
  if (typeof values === "object" && values !== null) {
    for (const [key, value] of Object.entries(values as Record<string, unknown>)) {
      if (typeof value === "string") catalogue[`${namespace}.${key}`] = value;
    }
  } else if (typeof values === "string") {
    catalogue[namespace] = values;
  }
}

export const i18n = {
  t(key: string, options?: { defaultValue?: string }): string {
    return catalogue[key] ?? options?.defaultValue ?? key;
  },
};
