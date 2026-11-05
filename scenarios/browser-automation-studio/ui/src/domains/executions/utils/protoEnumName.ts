import type { GenEnum } from '@bufbuild/protobuf/codegenv2';

export const enumShortName = <T extends number>(
  desc: GenEnum<T>,
  value: T | number,
): string | undefined => {
  const enumValue = desc.value[value as T];
  if (!enumValue) {
    return undefined;
  }

  const rawName = enumValue.name;
  const prefix = desc.sharedPrefix;
  const shortName =
    prefix && rawName.startsWith(prefix)
      ? rawName.slice(prefix.length)
      : enumValue.localName;

  return shortName.toLowerCase();
};
