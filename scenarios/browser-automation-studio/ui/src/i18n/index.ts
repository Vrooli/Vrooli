export {};

export const i18n = { t: (key: string, options?: { defaultValue?: string }) => options?.defaultValue ?? key };
