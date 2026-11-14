// Built as a self-contained inert source module, executed only inside the
// opaque-origin preview iframe. It has no access to host state or blob endpoints.
import { buildPresentation, parseZip, renderSlide, type PresentationData, type SlideHandle } from "@aiden0z/pptx-renderer";
import { publicationParagraphs, publicationPath, publicationXml, xmlChildren } from "./publicationXml";
import type { PresentationEvent, PresentationMetadata, PresentationView } from "./publicationData";

let presentation: PresentationData | null = null, metadata: PresentationMetadata | null = null, view: PresentationView | null = null;
let main: SlideHandle | null = null, observer: IntersectionObserver | null = null, generation = 0;
const thumbs = new Map<Element, SlideHandle>(), root = document.getElementById("presentation") ?? document.body;
const send = (event: PresentationEvent) => { parent.postMessage({ channel: "wc-presentation", ...event }, "*"); };
const clear = () => { main?.dispose(); main = null; observer?.disconnect(); observer = null; for (const h of thumbs.values()) h.dispose(); thumbs.clear(); root.replaceChildren(); };
function dispose() { generation++; clear(); presentation = null; metadata = null; }
function slideChange(index: number) {
  if (!view || !presentation || index < 0 || index >= presentation.slides.length) return;
  view = { ...view, index, thumbnails: root.clientWidth >= 720 && view.thumbnails }; send({ type: "slide", index }); draw();
  root.querySelector<HTMLElement>(".stage")?.focus({ preventScroll: true });
}

function scaled(index: number, host: HTMLElement, width: number): SlideHandle | null {
  if (!presentation || !presentation.slides[index]) return null;
  try {
    const h = renderSlide(presentation, presentation.slides[index], { externalMedia: "block", pdfjs: false, onNodeError: () => { send({ type: "warning" }); }, onNavigate: target => { if (target.slideIndex !== undefined) slideChange(target.slideIndex); } });
    h.element.style.transform = `scale(${String(width / presentation.width)})`; h.element.style.transformOrigin = "top left";
    host.style.width = `${String(width)}px`; host.style.height = `${String(width * presentation.height / presentation.width)}px`;
    host.append(h.element); return h;
  } catch { send({ type: "warning" }); return null; }
}
function draw() {
  if (!presentation || !metadata || !view) return;
  clear();
  const currentView = view, currentPresentation = presentation;
  const wide = document.documentElement.clientWidth >= 720, grid = view.thumbnails && !wide;
  document.documentElement.dir = view.rtl ? "rtl" : "ltr";
  document.documentElement.lang = view.language;
  document.title = view.labels.slide;
  for (const [key, value] of Object.entries(view.theme)) root.style.setProperty(`--${key}`, value);
  root.className = grid ? "grid-view" : wide && view.thumbnails ? "with-rail" : "";
  if (view.thumbnails) {
    const list = document.createElement("nav"); list.setAttribute("aria-label", view.labels.slide); list.className = "slides";
    observer = new IntersectionObserver(entries => {
      for (const entry of entries) {
        const button = entry.target as HTMLButtonElement, preview = button.querySelector<HTMLElement>(".thumbnail"); if (!preview) continue;
        if (entry.isIntersecting && !thumbs.has(button)) { const h = scaled(Number(button.dataset.index), preview, preview.clientWidth || 140); if (h) thumbs.set(button, h); }
        else if (!entry.isIntersecting && thumbs.has(button)) { thumbs.get(button)?.dispose(); thumbs.delete(button); preview.replaceChildren(); }
      }
    }, { root: list, rootMargin: "160px" });
    metadata.slides.forEach((slide, index) => {
      const button = document.createElement("button"); button.type = "button"; button.dataset.index = String(index); button.setAttribute("aria-label", `${currentView.labels.slide} ${String(index + 1)} · ${slide.title}`);
      if (index === currentView.index) button.setAttribute("aria-current", "true");
      button.onclick = () => { slideChange(index); };
      const image = document.createElement("span"); image.className = "thumbnail"; image.setAttribute("aria-hidden", "true"); image.style.aspectRatio = `${String(currentPresentation.width)}/${String(currentPresentation.height)}`; image.inert = true;
      const title = document.createElement("span"); title.className = "caption"; title.textContent = `${String(index + 1)} · ${slide.title}${slide.hidden ? ` · ${currentView.labels.hidden}` : ""}`;
      button.append(image, title); list.append(button); observer?.observe(button);
    });
    root.append(list);
  }
  if (!grid) {
    const stage = document.createElement("div"); stage.className = "stage"; stage.tabIndex = 0; stage.setAttribute("role", "region"); stage.setAttribute("aria-label", `${view.labels.slide} ${String(view.index + 1)} · ${(metadata.slides[view.index]?.title ?? "")}`);
    const availableWidth = Math.max(80, document.documentElement.clientWidth - (wide && view.thumbnails ? 184 : 0) - 24), availableHeight = Math.max(60, document.documentElement.clientHeight - 24);
    const fit = Math.min(availableWidth / presentation.width, availableHeight / presentation.height);
    const width = presentation.width * fit * view.zoom, canvas = document.createElement("div"); canvas.className = "canvas"; canvas.dir = "ltr";
    main = scaled(view.index, canvas, width); stage.append(canvas); root.append(stage);
  }
}

type RuntimeCommand = { channel: string; type: "hello" | "dispose" } | { channel: string; type: "view"; view: PresentationView } | { channel: string; type: "open"; bytes: ArrayBuffer; xml: Record<string, string> };
window.addEventListener("message", (event: MessageEvent<RuntimeCommand>) => {
  if (event.source !== parent || event.data.channel !== "wc-presentation") return;
  if (event.data.type === "hello") { send({ type: "boot" }); return; }
  if (event.data.type === "view") { view = event.data.view; draw(); }
  if (event.data.type === "dispose") dispose();
  if (event.data.type !== "open" || presentation) return;
  const input = event.data;
  const current = ++generation;
  void (async () => {
    try {
      const files = await parseZip(input.bytes, { maxEntries: 4000, maxEntryUncompressedBytes: 16 * 1024 * 1024, maxTotalUncompressedBytes: 64 * 1024 * 1024, maxMediaBytes: 48 * 1024 * 1024, maxConcurrency: 2 });
      if (current !== generation) return;
      const doc = publicationXml(files.presentation);
      if (xmlChildren(doc, "sldId").length > 300) throw new RangeError();
      const notes = input.xml;
      for (const source of [files.presentation, files.presentationRels, ...files.slides.values(), ...files.slideRels.values(), ...files.slideLayouts.values(), ...files.slideMasters.values(), ...files.themes.values(), ...files.charts.values(), ...Object.values(notes)]) publicationXml(source);
      presentation = buildPresentation(files, { lazySlides: true });
      if (!presentation.slides.length) throw Error("invalid");
      metadata = { width: presentation.width, height: presentation.height, warnings: false, slides: presentation.slides.map((slide, index) => {
        const source = files.slides.get(slide.slidePath) ?? slide.sourceXml ?? "", doc = publicationXml(source), paragraphs = publicationParagraphs(doc);
        const titleShape = xmlChildren(doc, "sp").find(shape => xmlChildren(shape, "ph").some(ph => ["title", "ctrTitle"].includes(ph.getAttribute("type") ?? "")));
        const base = slide.slidePath.slice(0, slide.slidePath.lastIndexOf("/") + 1), relName = `${base}_rels/${slide.slidePath.slice(base.length)}.rels`;
        let note = "";
        if (notes[relName]) {
          const rel = xmlChildren(publicationXml(notes[relName]), "Relationship").find(r => r.getAttribute("Type")?.endsWith("/notesSlide") && r.getAttribute("TargetMode") !== "External");
          const path = rel ? publicationPath(slide.slidePath, rel.getAttribute("Target") ?? "") : null;
          if (path && notes[path]) note = xmlChildren(publicationXml(notes[path]), "sp").filter(shape => xmlChildren(shape, "ph").some(ph => ph.getAttribute("type") === "body")).map(publicationParagraphs).join("\n\n");
        }
        return { title: (titleShape ? publicationParagraphs(titleShape) : paragraphs.split("\n")[0] || String(index + 1)).slice(0, 300), text: paragraphs.slice(0, 100000), notes: note.slice(0, 100000), hidden: slide.hidden ?? false };
      }) };
      if (![metadata.width, metadata.height].every(value => Number.isFinite(value) && value >= 1 && value <= 10000)) throw new RangeError();
      send({ type: "ready", metadata }); draw();
    } catch (error) { dispose(); send({ type: "error", error: error instanceof RangeError ? "limit" : "invalid" }); }
  })();
});
window.addEventListener("keydown", event => {
  if (event.altKey || event.ctrlKey || event.metaKey || (event.target instanceof Element && event.target.closest("input,textarea,[contenteditable]"))) return;
  const key = event.key;
  if (["ArrowRight", "ArrowLeft", "PageDown", "PageUp", "Home", "End"].includes(key) && view && presentation) {
    event.preventDefault();
    slideChange(key === "Home" ? 0 : key === "End" ? presentation.slides.length - 1 : view.index + ((key === "ArrowRight" || key === "PageDown") ? 1 : -1));
  }
});
// Deck hyperlinks/media stay inert. Explicit host controls own all navigation.
root.addEventListener("click", event => { if ((event.target as Element).closest("a,audio,video")) event.preventDefault(); }, true);
window.addEventListener("pagehide", dispose);
new ResizeObserver(draw).observe(document.documentElement);
send({ type: "boot" });
