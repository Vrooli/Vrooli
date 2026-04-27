const externalSchemes = /^(https?:\/\/|mailto:|tel:)/i;

export function isExternalHref(href: string): boolean {
  return externalSchemes.test(href.trim());
}

export function looksLikeFileReference(href: string): boolean {
  const value = href.trim();
  if (!value || isExternalHref(value) || value.startsWith("#")) {
    return false;
  }
  if (value.startsWith("file://")) {
    return true;
  }
  if (value.startsWith("/")) {
    return hasFileLikeShape(value);
  }
  return hasFileLikeShape(value);
}

function hasFileLikeShape(value: string): boolean {
  const withoutLine = value.replace(/:\d+$/, "");
  return (
    withoutLine.includes("/") ||
    withoutLine.includes("\\") ||
    /\.[a-z0-9]+$/i.test(withoutLine)
  );
}
