// A large document can embed over a hundred full-resolution inline images. Even
// with `content-visibility` (which several mobile web views support poorly or
// not at all), keeping every decoded image alive exhausts memory on a phone and
// forces constant re-decoding while scrolling — the dominant cost of scrolling
// an image-heavy markdown file.
//
// The remedy is to window the images: mount an <img> only while it is near the
// viewport and drop it (reserving its space) once it scrolls away, so only a
// handful are decoded at any moment. Every image shares a single
// IntersectionObserver rather than owning one. Where IntersectionObserver is
// unavailable (jsdom, very old engines) callers fall back to treating the image
// as visible, matching the prior eager behavior.

type NearCallback = (near: boolean) => void;

const ROOT_MARGIN = "300px";

let nearObserver: IntersectionObserver | null = null;
const nearCallbacks = new Map<Element, NearCallback>();

function ensureNear(): IntersectionObserver | null {
  if (typeof IntersectionObserver === "undefined") return null;
  nearObserver ??= new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        nearCallbacks.get(entry.target)?.(entry.isIntersecting);
      }
    },
    { rootMargin: ROOT_MARGIN },
  );
  return nearObserver;
}

/** Calls `onChange(near)` whenever `element` enters or leaves the viewport
 *  margin, so a caller can mount heavy content only while it is near. */
export function observeNear(element: Element, onChange: NearCallback): () => void {
  const shared = ensureNear();
  if (!shared) {
    onChange(true);
    return () => {};
  }
  nearCallbacks.set(element, onChange);
  shared.observe(element);
  return () => {
    nearCallbacks.delete(element);
    shared.unobserve(element);
  };
}
