import { useState, type MouseEvent } from "react";
import { Link } from "react-router-dom";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useJobProgress } from "../jobs/useJobProgress";
import { useObjectMask, type ObjectMaskClient } from "./useObjectMask";

/** Prompted object regions share Select; colour-based selection remains an
 * explicitly chosen tool. This surface never silently substitutes a heuristic. */
export function ObjectMaskEditor({ image, imageUrl, client }: {
  image: File;
  imageUrl: string;
  client?: ObjectMaskClient;
}) {
  const { t } = useTranslation();
  const s = useObjectMask(image, client);
  const progress = useJobProgress(s.jobId, s.busy);
  const [negative, setNegative] = useState(false);
  const [x, setX] = useState("50");
  const [y, setY] = useState("50");
  const k = strings.select.objectMask;
  const fieldClass = "rounded-control border border-app-border bg-app-surface p-2";
  const buttonClass = "rounded-control border border-app-border px-3 py-2 disabled:opacity-50";

  const addCoordinate = () => {
    const nx = Number(x) / 100;
    const ny = Number(y) / 100;
    if (x.trim() && y.trim() && Number.isFinite(nx) && Number.isFinite(ny)
      && nx >= 0 && nx <= 1 && ny >= 0 && ny <= 1) {
      s.addPoint({ x: nx, y: ny, negative });
    }
  };
  const clickImage = (event: MouseEvent<HTMLButtonElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    if (rect.width && rect.height) {
      s.addPoint({
        x: Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width)),
        y: Math.max(0, Math.min(1, (event.clientY - rect.top) / rect.height)),
        negative,
      });
    }
  };

  return (
    <div className="grid gap-6 md:grid-cols-[2fr_1fr]">
      <div className="flex min-w-0 flex-col gap-3">
        <p className="text-sm text-app-muted-foreground">{t(k.hint)}</p>
        <button
          type="button"
          onClick={clickImage}
          aria-label={t(k.imageAction)}
          className="relative w-fit max-w-full cursor-crosshair overflow-hidden rounded-card border border-app-border"
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              addCoordinate();
            }
          }}
        >
          <img src={imageUrl} alt={t(strings.select.imageAlt)} className="block max-h-[60vh] max-w-full" draggable={false} />
          {s.maskUrl ? (
            <img src={s.maskUrl} alt={t(strings.select.maskAlt)} className="pointer-events-none absolute inset-0 h-full w-full opacity-50 mix-blend-screen" />
          ) : null}
          {s.points.map((point, index) => (
            <span
              key={index}
              aria-hidden="true"
              style={{ left: `${point.x * 100}%`, top: `${point.y * 100}%` }}
              className={`pointer-events-none absolute flex h-5 w-5 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full border-2 border-white text-xs font-bold shadow ${point.negative ? "bg-app-danger text-white" : "bg-app-primary text-app-primary-foreground"}`}
            >
              {point.negative ? "−" : "+"}
            </span>
          ))}
        </button>
        <div className="flex flex-wrap items-end gap-3">
          <label className="flex flex-col text-sm">
            {t(k.pointKind)}
            <select value={negative ? "exclude" : "include"} onChange={(event) => setNegative(event.target.value === "exclude")} className={fieldClass}>
              <option value="include">{t(k.include)}</option>
              <option value="exclude">{t(k.exclude)}</option>
            </select>
          </label>
          <label className="flex flex-col text-sm">
            {t(strings.select.pointX)}
            <input type="number" min={0} max={100} value={x} onChange={(event) => setX(event.target.value)} className={`w-20 ${fieldClass}`} />
          </label>
          <label className="flex flex-col text-sm">
            {t(strings.select.pointY)}
            <input type="number" min={0} max={100} value={y} onChange={(event) => setY(event.target.value)} className={`w-20 ${fieldClass}`} />
          </label>
          <button type="button" onClick={addCoordinate} className={buttonClass}>{t(k.addPoint)}</button>
          <button type="button" disabled={!s.points.length} onClick={s.undo} className={buttonClass}>{t(k.undo)}</button>
          <button type="button" disabled={!s.points.length} onClick={s.clear} className={buttonClass}>{t(k.clear)}</button>
        </div>
      </div>
      <div className="flex flex-col items-start gap-4">
        <label className="flex w-full flex-col gap-1 text-sm">
          {t(k.model)}
          <select value={s.model} onChange={(event) => s.chooseModel(event.target.value)} className={fieldClass}>
            <option value="">{t(k.automatic)}</option>
            <option value="mobilesam">{t(k.mobileSam)}</option>
            <option value="sam-2.1-tiny">{t(k.sam2Tiny)}</option>
          </select>
        </label>
        <p className="text-sm text-app-muted-foreground">{t(k.disclosure)}</p>
        <button
          type="button"
          disabled={s.busy || !s.points.some((point) => !point.negative)}
          onClick={() => void s.run()}
          className="rounded-control bg-app-primary px-4 py-2 font-medium text-app-primary-foreground disabled:opacity-50"
        >
          {t(k.run)}
        </button>
        {s.busy ? (
          <>
            <p role="status">{t(k.running, { percent: progress?.jobId === s.jobId ? progress.progress : 0 })}</p>
            <button type="button" onClick={s.cancel} className={buttonClass}>{t(k.cancel)}</button>
          </>
        ) : null}
        {s.error ? (
          <div role="alert" className="text-sm text-app-danger">
            <p>{s.error}</p>
            <Link to="/models" className="underline">{t(k.manageModels)}</Link>
          </div>
        ) : null}
        {s.cancelError ? <p role="alert" className="text-sm text-app-danger">{t(k.cancelFailed, { message: s.cancelError })}</p> : null}
        {s.maskUrl ? (
          <>
            <p role="status" className="text-sm">{t(k.ready, { model: s.servingModel })}</p>
            <a href={s.maskUrl} download="object-region.png" className={buttonClass}>{t(k.download)}</a>
          </>
        ) : null}
      </div>
    </div>
  );
}
