import { useEffect, useState } from "react";

import { fetchItemBlob } from "../../api/transfer";

interface ItemThumbnailProps {
  itemId: string;
  alt: string;
}

/**
 * Render an item's server-generated thumbnail. The thumbnail endpoint is
 * device-token-authed, so a plain `<img src>` can't reach it — we fetch the
 * blob (with the token attached by `authedFetch`) into an object URL, and
 * revoke it on unmount / id change. Renders nothing until the blob resolves.
 */
export function ItemThumbnail({ itemId, alt }: ItemThumbnailProps) {
  const [src, setSrc] = useState<string | null>(null);

  useEffect(() => {
    let revoked = false;
    let objectUrl: string | null = null;
    fetchItemBlob(itemId, { thumb: true })
      .then((blob) => {
        if (revoked) return;
        objectUrl = URL.createObjectURL(blob);
        setSrc(objectUrl);
      })
      .catch(() => {
        // Thumbnail is decorative; a fetch failure just leaves the icon fallback.
      });
    return () => {
      revoked = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [itemId]);

  if (!src) {
    return <span className="h-10 w-10 shrink-0 rounded-control bg-app-surface-muted" aria-hidden="true" />;
  }
  return (
    <img
      src={src}
      alt={alt}
      className="h-10 w-10 shrink-0 rounded-control object-cover"
    />
  );
}
