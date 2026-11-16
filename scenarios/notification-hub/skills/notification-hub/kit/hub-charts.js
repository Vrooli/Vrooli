/*
 * Notification Hub chart kit: the script. Exposes window.HubCharts.
 * Source of truth for every template in kit/templates/. tools/sync.mjs copies it
 * into each template; never edit the copy inside a template.
 *
 * Plain ES2019 (no optional chaining), no dependencies, nothing loaded from the
 * network. Every chart is drawn at its container's real width (the SVG viewBox
 * equals the measured pixel width), so 11 px labels stay 11 px on a phone, and
 * redrawn when the width changes. Charts draw only the numbers they are given;
 * none is invented, interpolated beyond the data, or filled in for a missing source.
 *
 * Page API (Notification Hub host): hub.answer.set(key, label, value),
 * hub.answer.remove(key), hub.context, hub.prefill(questionId, value). All are
 * optional: outside the Hub, or in a Hub without hub.answer, pages stay
 * read-only and still preview every choice.
 */
(function () {
  "use strict";
  if (window.HubCharts) return;

  var NS = "http://www.w3.org/2000/svg";
  var ZONE = undefined;
  var LOCALE = "en-GB";

  /* ---------- DOM helpers ---------- */
  function esc(value) {
    return String(value === undefined || value === null ? "" : value)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
  function h(tag, attrs, parent) {
    var e = document.createElement(tag);
    for (var k in attrs || {}) {
      var v = attrs[k];
      if (v === undefined || v === null || v === false) continue;
      if (k === "text") e.textContent = v;
      else if (k === "html") e.innerHTML = v;
      else if (k === "class") e.className = v;
      else e.setAttribute(k, v === true ? "" : v);
    }
    if (parent) parent.appendChild(e);
    return e;
  }
  function s(tag, attrs, parent) {
    var e = document.createElementNS(NS, tag);
    for (var k in attrs || {}) {
      var v = attrs[k];
      if (v === undefined || v === null) continue;
      if (k === "text") e.textContent = v;
      else e.setAttribute(k, v);
    }
    if (parent) parent.appendChild(e);
    return e;
  }
  function clear(el) { while (el.firstChild) el.removeChild(el.firstChild); return el; }
  function byId(id) { return document.getElementById(id); }
  function num(v) { return typeof v === "number" && isFinite(v); }
  function clamp(v, lo, hi) { return Math.max(lo, Math.min(hi, v)); }
  function ms(t) { if (typeof t === "number") return t; var v = Date.parse(t); return isNaN(v) ? NaN : v; }
  function truncate(text, max) { text = String(text); return text.length > max ? text.slice(0, Math.max(1, max - 1)) + "…" : text; }
  function plural(n, one, many) { return n === 1 ? one : (many || one + "s"); }

  /* ---------- tones ---------- */
  var NAMED = { accent: "var(--kc-accent)", good: "var(--hub-success)", warn: "var(--hub-warning)", crit: "var(--hub-danger)", muted: "var(--hub-muted)", ink: "var(--hub-fg)", line: "var(--kc-line)" };
  function tone(t) {
    if (typeof t === "number" && t >= 1 && t <= 8) return "var(--kc-tone-" + Math.round(t) + ")";
    if (typeof t === "string" && NAMED[t]) return NAMED[t];
    return "var(--kc-tone-7)";
  }

  /* ---------- time formatting (in the page's time zone) ---------- */
  var fmtCache = {};
  function dtf(key, opts) {
    var id = key + "|" + (ZONE || "");
    if (!fmtCache[id]) {
      var o = {};
      for (var k in opts) o[k] = opts[k];
      if (ZONE) o.timeZone = ZONE;
      try { fmtCache[id] = new Intl.DateTimeFormat(LOCALE, o); } catch (e) { delete o.timeZone; fmtCache[id] = new Intl.DateTimeFormat(LOCALE, o); }
    }
    return fmtCache[id];
  }
  function parts(t) {
    var out = {};
    dtf("parts", { weekday: "short", day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", hourCycle: "h23" })
      .formatToParts(new Date(ms(t))).forEach(function (p) { out[p.type] = p.value; });
    out.hour = out.hour === "24" ? "00" : out.hour;
    return out;
  }
  var fmt = {
    hm: function (t) { var p = parts(t); return p.hour + ":" + p.minute; },
    weekday: function (t) { return parts(t).weekday; },
    when: function (t) { var p = parts(t); return p.weekday + " " + p.hour + ":" + p.minute; },
    day: function (t) { var p = parts(t); return p.weekday + " " + p.day; },
    date: function (t) { var p = parts(t); return p.day + " " + p.month; },
    dayDate: function (t) { var p = parts(t); return p.weekday + " " + p.day + " " + p.month; },
    full: function (t) { var p = parts(t); return p.weekday + " " + p.day + " " + p.month + " · " + p.hour + ":" + p.minute; },
    hours: function (hrs) {
      if (!num(hrs)) return "–";
      if (hrs === 0) return "0 h";
      if (hrs < 1) return Math.round(hrs * 60) + " min";
      if (hrs < 48) return Math.round(hrs) + " h";
      var d = Math.floor(hrs / 24), r = Math.round(hrs - d * 24);
      return d + "d " + r + "h";
    },
    pct: function (v, digits) { return num(v) ? v.toFixed(digits || 0) + "%" : "–"; },
    share: function (v) { return num(v) ? Math.round(v * 100) + "%" : "–"; },
    n: function (v, digits) { return num(v) ? (digits ? v.toFixed(digits) : Math.round(v).toLocaleString(LOCALE)) : "–"; },
    money: function (v, currency) {
      if (!num(v)) return "–";
      try { return new Intl.NumberFormat(currency && currency !== "USD" ? LOCALE : "en-US", { style: "currency", currency: currency || "USD", maximumFractionDigits: 0 }).format(v); } catch (e) { return "$" + Math.round(v); }
    }
  };
  function hourOf(t) { return +parts(t).hour; }
  /** Local midnights between a and b (ms), in the page's zone. */
  function dayStarts(a, b) {
    var out = [], t = Math.ceil(a / 3600e3) * 3600e3, guard = 0;
    while (t <= b && guard++ < 24 * 400) {
      var p = parts(t);
      if (p.hour === "00" && p.minute === "00") { out.push(t); t += 23 * 3600e3; } else t += 3600e3;
    }
    return out;
  }

  /* ---------- tooltip ---------- */
  var tipEl = null;
  function tipNode() {
    if (!tipEl) { tipEl = h("div", { class: "kc-tip", role: "status", "aria-live": "polite" }); tipEl.hidden = true; document.body.appendChild(tipEl); }
    return tipEl;
  }
  function showTipAt(html, x, y) {
    var t = tipNode();
    t.innerHTML = html; t.hidden = false;
    var r = t.getBoundingClientRect(), vw = document.documentElement.clientWidth;
    var sx = window.pageXOffset || 0, sy = window.pageYOffset || 0;
    var L = x + 14, T = y + 16;
    if (L + r.width > sx + vw - 6) L = x - r.width - 14;
    if (L < sx + 6) L = sx + 6;
    if (T - sy < 6) T = sy + 6;
    t.style.left = Math.round(L) + "px"; t.style.top = Math.round(T) + "px";
  }
  function showTip(html, ev) { showTipAt(html, ev.clientX + (window.pageXOffset || 0), ev.clientY + (window.pageYOffset || 0)); }
  function hideTip() { if (tipEl) tipEl.hidden = true; }
  function bindTip(el, html) {
    var get = function () { return typeof html === "function" ? html() : html; };
    el.addEventListener("pointermove", function (e) { showTip(get(), e); });
    el.addEventListener("pointerdown", function (e) { showTip(get(), e); });
    el.addEventListener("pointerleave", function (e) { if (e.pointerType === "mouse") hideTip(); });
    el.addEventListener("focus", function () { var r = el.getBoundingClientRect(); showTipAt(get(), r.left + (window.pageXOffset || 0) + r.width / 2, r.bottom + (window.pageYOffset || 0) - 10); });
    el.addEventListener("blur", hideTip);
  }
  document.addEventListener("pointerdown", function (e) {
    if (!tipEl || tipEl.hidden) return;
    var t = e.target;
    if (t && t.closest && t.closest(".kc-chart, .kc-prop, .kc-strip__bar, [data-kc-tip]")) return;
    hideTip();
  }, true);

  /* ---------- sizing ---------- */
  /** Calls fn(width) now and whenever the element's width changes. */
  function onWidth(el, fn) {
    var last = -1;
    function run() {
      var w = Math.round(el.clientWidth);
      if (w > 0 && w !== last) { last = w; fn(w); }
    }
    if (typeof ResizeObserver === "function") new ResizeObserver(run).observe(el);
    window.addEventListener("resize", run);
    run();
    return { redraw: function () { last = -1; run(); } };
  }
  function svgRoot(el, W, H, label) {
    clear(el);
    return s("svg", { viewBox: "0 0 " + W + " " + H, width: W, height: H, role: "img", "aria-label": label || el.getAttribute("aria-label") || "" }, el);
  }
  /** Shifts an SVG text so it stays inside [minX, maxX]. */
  function fitText(t, minX, maxX) {
    var len = 0;
    try { len = t.getComputedTextLength(); } catch (e) { return t; }
    if (!len) return t;
    var x = +t.getAttribute("x"), a = t.getAttribute("text-anchor") || "start";
    var left = a === "end" ? x - len : a === "middle" ? x - len / 2 : x;
    if (left + len > maxX) left = maxX - len;
    if (left < minX) left = minX;
    t.setAttribute("text-anchor", "start"); t.setAttribute("x", left);
    return t;
  }
  function niceMax(v) {
    if (!(v > 0)) return 1;
    var p = Math.pow(10, Math.floor(Math.log10(v))), f = v / p;
    return (f <= 1 ? 1 : f <= 2 ? 2 : f <= 2.5 ? 2.5 : f <= 5 ? 5 : 10) * p;
  }

  /* ---------- page data, answer, context ---------- */
  function data() {
    var el = byId("hub-data");
    if (!el) return null;
    try { var d = JSON.parse(el.textContent || "null"); return d && typeof d === "object" ? d : null; } catch (e) { return null; }
  }
  function hub() { return window.hub || null; }
  var answer = {
    /** True when the Hub records page changes (hub.answer). */
    live: function () { var x = hub(); return !!(x && x.answer && typeof x.answer.set === "function"); },
    /** True when the Hub can at least prefill its form (hub.prefill). */
    canPrefill: function () { var x = hub(); return !!(x && typeof x.prefill === "function"); },
    set: function (key, label, value) {
      if (!answer.live()) return false;
      try { hub().answer.set(key, label, value); return true; } catch (e) { return false; }
    },
    remove: function (key) {
      var x = hub();
      if (!x || !x.answer || typeof x.answer.remove !== "function") return false;
      try { x.answer.remove(key); return true; } catch (e) { return false; }
    },
    prefill: function (questionId, value) {
      if (!answer.canPrefill()) return false;
      try { hub().prefill(questionId, value); return true; } catch (e) { return false; }
    }
  };
  function context() { var x = hub(); return x && x.context && typeof x.context === "object" ? x.context : null; }
  function onContext(fn) { window.addEventListener("hubcontext", function () { fn(context()); }); }
  /** One line under the controls that says whether choices are recorded. */
  function answerNote(el, kind) {
    if (!el) return;
    if (answer.live()) el.textContent = kind === "status" ? "Changes you make here go into your reply. Send it, or just mark this read." : "Your choices here go into your answer. Send it from the bar below.";
    else if (answer.canPrefill()) el.textContent = kind === "status" ? "This Hub can't record page choices yet; write your steer in a reply." : "“Choose this” fills your answer below; other changes here are a preview.";
    else el.textContent = "Viewing outside Notification Hub: choices here preview the effect only; nothing is sent.";
  }

  /* ---------- provenance badge and unavailable sections ---------- */
  var SRC_WORDS = { real: "Real data", partial: "Partly real", unavailable: "Unavailable" };
  /** True when a section has data and its source is not marked unavailable. */
  function available(section) { return !!(section && typeof section === "object" && !(section.source && section.source.status === "unavailable")); }
  function badge(source) {
    var st = source && SRC_WORDS[source.status] ? source.status : "unavailable";
    var text = SRC_WORDS[st] + (source && source.label ? " · " + source.label : "");
    return h("span", { class: "kc-src is-" + st, text: text, title: source && source.note ? source.note : text });
  }
  /** Fills a section's [data-src] slot; returns whether the section can draw. */
  function mountSource(sectionEl, section, missingLabel) {
    var slot = sectionEl ? sectionEl.querySelector("[data-src]") : null;
    var src = section && section.source ? section.source : (section ? { status: "real" } : { status: "unavailable", label: missingLabel || "no data sent" });
    if (slot) { clear(slot); slot.appendChild(badge(src)); slot.removeAttribute("hidden"); }
    return available(section);
  }
  function unavailable(el, section, what) {
    clear(el);
    var src = section && section.source ? section.source : null;
    var why = String(src && src.note ? src.note : (src && src.label ? src.label : "the producer sent no data for it")).replace(/[.\s]+$/, "");
    why = why.charAt(0).toUpperCase() + why.slice(1);
    var box = h("div", { class: "kc-unavailable", role: "note" }, el);
    box.innerHTML = "<span><b>" + esc(what || "This section") + ": unavailable.</b> " + esc(why) + ". Nothing is estimated in its place.</span>";
    return box;
  }
  /** Sets text of [data-text="path"] elements from data (dot path). */
  function bindText(root, d) {
    var els = root.querySelectorAll("[data-text]");
    for (var i = 0; i < els.length; i++) {
      var v = path(d, els[i].getAttribute("data-text"));
      if (typeof v === "string" || typeof v === "number") els[i].textContent = String(v);
      else if (els[i].hasAttribute("data-optional")) els[i].hidden = true;
    }
  }
  function path(obj, p) {
    var cur = obj, keys = String(p).split(".");
    for (var i = 0; i < keys.length; i++) { if (!cur || typeof cur !== "object") return undefined; cur = cur[keys[i]]; }
    return cur;
  }

  /* ---------- small blocks ---------- */
  function legend(el, items) {
    clear(el);
    (items || []).forEach(function (it) {
      var sp = h("span", {}, el), sw = h("i", { class: "kc-sw" + (it.swatch ? " is-" + it.swatch : "") }, sp);
      if (it.color && it.swatch !== "dash" && it.swatch !== "ring" && it.swatch !== "hatch" && it.swatch !== "band") sw.style.background = it.color;
      if (it.swatch === "ring" && it.color) sw.style.boxShadow = "inset 0 0 0 2px " + it.color;
      sp.appendChild(document.createTextNode(it.label));
      if (it.value !== undefined && it.value !== null) { sp.appendChild(document.createTextNode(" ")); h("b", { text: it.value }, sp); }
    });
    return el;
  }
  function tiles(el, list) {
    clear(el);
    (list || []).forEach(function (t) {
      var d = h("div", { class: "kc-tile" + (t.tone ? " is-" + t.tone : "") }, el);
      var big = h("span", { class: "kc-tile__big", text: t.value === undefined || t.value === null ? "–" : String(t.value) }, d);
      if (t.unit) h("small", { text: t.unit }, big);
      h("p", { text: t.text || "" }, d);
    });
    return el;
  }
  function stats(el, list) {
    clear(el);
    (list || []).forEach(function (t) {
      var d = h("div", { class: "kc-stat" }, el);
      h("span", { class: "kc-stat__lbl", text: t.label }, d);
      var big = h("span", { class: "kc-stat__big", text: t.value === undefined || t.value === null ? "–" : String(t.value) }, d);
      if (t.unit) h("small", { text: t.unit }, big);
      if (t.sub) h("span", { class: "kc-stat__lbl", text: t.sub }, d);
    });
    return el;
  }
  function pill(text, t) { return h("span", { class: "kc-pill" + (t ? " is-" + t : ""), text: text }); }
  /** A segmented control of buttons; onChange(value). Returns {set(value)}. */
  function seg(el, items, current, onChange, opts) {
    clear(el);
    el.classList.add(opts && opts.chips ? "kc-chips" : "kc-seg");
    el.setAttribute("role", "group");
    var buttons = (items || []).map(function (it) {
      var b = h("button", { type: "button", class: opts && opts.chips ? "kc-chip" : null, "data-v": it.value, "aria-pressed": String(it.value === current), text: it.label, title: it.title }, el);
      b.addEventListener("click", function () { api.set(it.value); onChange(it.value); });
      return b;
    });
    var api = { set: function (v) { buttons.forEach(function (b) { b.setAttribute("aria-pressed", String(b.getAttribute("data-v") === String(v))); }); } };
    return api;
  }

  /* ---------- burn chart ---------- */
  /*
   * spec: { start, end, now (ISO), readings:[{t,used}], lowPower:[{from,to,label?}],
   *   safePace: bool, series:[{key,label,points:[{t,used,lo?,hi?}], runOut?:ISO, runOutRange?:{early,late}}],
   *   selected, showAll (draw the other series faintly), height, endLabel ("Reset"), yLabel }
   */
  function burnChart(el, spec) {
    var selected = spec.selected;
    function series(key) { var l = spec.series || []; for (var i = 0; i < l.length; i++) if (l[i].key === key) return l[i]; return l[0] || null; }
    function draw(W) {
      var narrow = W < 520, Ht = spec.height || (narrow ? 236 : 284), m = { l: 38, r: 12, t: 24, b: 28 };
      var t0 = ms(spec.start), t1 = ms(spec.end), now = ms(spec.now);
      var x = function (t) { return m.l + ((ms(t) - t0) / (t1 - t0)) * (W - m.l - m.r); };
      var y0 = spec.yMin || 0;
      var y = function (u) { return m.t + (1 - (clamp(u, y0, 100) - y0) / (100 - y0)) * (Ht - m.t - m.b); };
      var svg = svgRoot(el, W, Ht);
      var sel = series(selected), readings = spec.readings || [];
      var last = readings.length ? readings[readings.length - 1] : null;
      var nowUsed = last ? last.used : null;
      (spec.lowPower || []).concat(sel && sel.lowSpans ? sel.lowSpans : []).forEach(function (span) {
        var a = Math.max(t0, ms(span.from)), b = Math.min(t1, ms(span.to));
        if (b > a) s("rect", { x: x(a), y: m.t, width: x(b) - x(a), height: Ht - m.t - m.b, fill: "var(--hub-surface-muted)" }, svg);
      });
      if (sel && sel.runOut && ms(sel.runOut) < t1) s("rect", { x: x(sel.runOut), y: m.t, width: x(t1) - x(sel.runOut), height: Ht - m.t - m.b, fill: "var(--kc-crit-soft)" }, svg);
      (y0 ? [y0, (y0 + 100) / 2, 100] : [0, 25, 50, 75, 100]).forEach(function (g) {
        s("line", { x1: m.l, x2: W - m.r, y1: y(g), y2: y(g), class: "kc-grid" }, svg);
        s("text", { x: m.l - 6, y: y(g) + 4, "text-anchor": "end", text: g + "%" }, svg);
      });
      var days = dayStarts(t0, t1), minGap = narrow ? 58 : 46, lastX = -1e9;
      days.forEach(function (d) {
        s("line", { x1: x(d), x2: x(d), y1: Ht - m.b, y2: Ht - m.b + 4, stroke: "var(--kc-line)" }, svg);
        if (x(d) - lastX >= minGap && x(d) < W - m.r - 24) { s("text", { x: x(d) + 3, y: Ht - m.b + 17, text: fmt.day(d) }, svg); lastX = x(d); }
      });
      var firstLow = (spec.lowPower || []).filter(function (sp) { return ms(sp.from) < now && ms(sp.to) > t0; })[0];
      if (firstLow && x(Math.min(now, ms(firstLow.to))) - x(Math.max(t0, ms(firstLow.from))) > 70) s("text", { x: x(Math.max(t0, ms(firstLow.from))) + 5, y: m.t + 14, class: "kc-halo", text: firstLow.label || "Low power on" }, svg);
      function line(points, attrs) {
        if (!points.length) return null;
        attrs.points = points.map(function (p) { return x(p.t) + "," + y(p.used); }).join(" ");
        attrs.fill = "none"; attrs["stroke-linejoin"] = "round"; attrs["stroke-linecap"] = "round";
        return s("polyline", attrs, svg);
      }
      var lead = last ? [{ t: last.t, used: last.used, lo: last.used, hi: last.used }] : [];
      if (spec.showAll) (spec.series || []).forEach(function (sr) { if (sr !== sel) line(lead.concat(sr.points || []), { stroke: "var(--kc-faint)", "stroke-width": 1.25, "stroke-dasharray": "3 3" }); });
      if (sel && sel.points && sel.points.length && num(sel.points[0].lo) && num(sel.points[0].hi)) {
        var pts = lead.concat(sel.points);
        var band = pts.map(function (p) { return x(p.t) + "," + y(num(p.hi) ? p.hi : p.used); }).concat(pts.slice().reverse().map(function (p) { return x(p.t) + "," + y(num(p.lo) ? p.lo : p.used); }));
        s("polygon", { points: band.join(" "), fill: "var(--kc-accent-soft)", stroke: "var(--kc-accent)", "stroke-opacity": .35, "stroke-width": 1 }, svg);
      }
      if (spec.safePace && last && now < t1) s("line", { x1: x(last.t), y1: y(last.used), x2: x(t1), y2: y(100), stroke: "var(--hub-muted)", "stroke-width": 1.5, "stroke-dasharray": "5 5" }, svg);
      if (readings.length) {
        var rd = readings.map(function (p) { return x(p.t) + "," + y(p.used); }).join(" ");
        s("polygon", { points: x(readings[0].t) + "," + y(y0) + " " + rd + " " + x(last.t) + "," + y(y0), fill: "var(--hub-fg)", "fill-opacity": .06 }, svg);
        line(readings, { stroke: "var(--hub-fg)", "stroke-width": 2.25 });
      }
      if (sel) line(lead.concat(sel.points || []), { stroke: "var(--kc-accent)", "stroke-width": 2.5 });
      // markers: reset and now, labels kept apart
      s("line", { x1: x(t1), x2: x(t1), y1: m.t - 8, y2: Ht - m.b, stroke: "var(--hub-fg)", "stroke-width": 1.5 }, svg);
      s("text", { x: x(t1) - 4, y: m.t - 11, "text-anchor": "end", class: "kc-t-strong kc-halo", text: spec.endLabel || "Reset" }, svg);
      if (now > t0 && now < t1) {
        s("line", { x1: x(now), x2: x(now), y1: m.t - 8, y2: Ht - m.b, stroke: "var(--hub-fg)", "stroke-width": 1, "stroke-dasharray": "2 3" }, svg);
        if (x(t1) - x(now) > 64) s("text", { x: x(now), y: m.t - 11, "text-anchor": "middle", class: "kc-t-strong kc-halo", text: "Now" }, svg);
      }
      if (last) s("circle", { cx: x(last.t), cy: y(last.used), r: 5, fill: "var(--hub-fg)", stroke: "var(--hub-surface)", "stroke-width": 2 }, svg);
      if (sel && sel.points && sel.points.length) {
        var out = sel.runOut && ms(sel.runOut) <= t1, endP = sel.points[sel.points.length - 1];
        var ex = out ? x(sel.runOut) : x(endP.t), ey = out ? y(100) : y(endP.used);
        s("circle", { cx: ex, cy: ey, r: 5.5, fill: out ? "var(--hub-danger)" : "var(--kc-accent)", stroke: "var(--hub-surface)", "stroke-width": 2 }, svg);
        var lbl = out ? "Runs out " + fmt.when(sel.runOut) : Math.round(endP.used) + "% at " + (spec.endLabel || "reset").toLowerCase();
        var anchorEnd = ex > W / 2;
        fitText(s("text", { x: out ? ex + 8 : anchorEnd ? ex - 9 : ex + 9, y: out ? ey + 20 : ey - 12, "text-anchor": out ? "start" : anchorEnd ? "end" : "start", class: "kc-t-strong kc-halo", text: lbl }, svg), m.l + 4, W - 2);
      }
      // crosshair
      var cross = s("g", { opacity: 0 }, svg);
      var cl = s("line", { y1: m.t, y2: Ht - m.b, stroke: "var(--kc-faint)", "stroke-width": 1 }, cross);
      var cd = s("circle", { r: 4.5, fill: "var(--hub-surface)", stroke: "var(--hub-fg)", "stroke-width": 2 }, cross);
      var hit = s("rect", { x: m.l, y: 0, width: W - m.l - m.r, height: Ht, fill: "transparent", class: "kc-hit" }, svg);
      function nearest(list, t) { var best = null; for (var i = 0; i < list.length; i++) if (!best || Math.abs(ms(list[i].t) - t) < Math.abs(ms(best.t) - t)) best = list[i]; return best; }
      function move(e) {
        var r = svg.getBoundingClientRect(), px = (e.clientX - r.left) * (W / r.width);
        var t = t0 + clamp((px - m.l) / (W - m.l - m.r), 0, 1) * (t1 - t0), html, u;
        var lastT = last ? ms(last.t) : t0;
        if (t <= lastT && readings.length) {
          var p = nearest(readings, t); u = p.used;
          var low = (spec.lowPower || []).some(function (sp) { return ms(p.t) >= ms(sp.from) && ms(p.t) <= ms(sp.to); });
          html = "<b>" + esc(fmt.when(p.t)) + "</b><br><span class=\"kc-num\">" + esc(p.used.toFixed(1)) + "% used</span>" + (low ? "<br>Low power" : "");
          t = ms(p.t);
        } else if (sel && sel.points && sel.points.length) {
          var q = nearest(sel.points, t); u = q.used; t = ms(q.t);
          html = "<b>" + esc(fmt.when(q.t)) + "</b> · projected<br><span class=\"kc-num\">" + Math.round(q.used) + "%" + (num(q.lo) && num(q.hi) ? " (range " + Math.round(q.lo) + "–" + Math.round(q.hi) + "%)" : "") + "</span>";
        } else { hideTip(); return; }
        cross.setAttribute("opacity", 1); cl.setAttribute("x1", x(t)); cl.setAttribute("x2", x(t)); cd.setAttribute("cx", x(t)); cd.setAttribute("cy", y(u));
        showTip(html, e);
      }
      hit.addEventListener("pointermove", move);
      hit.addEventListener("pointerdown", move);
      hit.addEventListener("pointerleave", function () { cross.setAttribute("opacity", 0); hideTip(); });
    }
    var sizer = onWidth(el, draw);
    return { select: function (key) { selected = key; sizer.redraw(); } };
  }

  /* ---------- mode strips: what each option does over time ---------- */
  /* spec: { start, end, rows:[{key,label,segments:[{from,to,mode:"full"|"low"|"stopped"}]}], selected, onSelect, modeLabels } */
  function modeStrips(el, axisEl, spec) {
    var LBL = spec.modeLabels || { full: "Full power", low: "Low power", stopped: "Stopped" };
    var t0 = ms(spec.start), t1 = ms(spec.end);
    clear(el);
    var rows = (spec.rows || []).map(function (r) {
      var row = h("div", { class: "kc-strip", tabindex: "0", role: "button", "data-k": r.key, "aria-label": "Preview " + r.label }, el);
      h("div", { class: "kc-strip__nm", text: r.label }, row);
      var bar = h("div", { class: "kc-strip__bar" }, row);
      (r.segments || []).forEach(function (sg) {
        var a = Math.max(t0, ms(sg.from)), b = Math.min(t1, ms(sg.to));
        if (!(b > a)) return;
        var i = h("i", { class: "kc-mode-" + sg.mode }, bar);
        i.style.flex = String(b - a);
        bindTip(i, "<b>" + esc(LBL[sg.mode] || sg.mode) + "</b><br><span class=\"kc-num\">" + esc(fmt.when(a)) + " → " + esc(fmt.when(b)) + " · " + esc(fmt.hours((b - a) / 3600e3)) + "</span>");
      });
      function pick() { if (spec.onSelect) spec.onSelect(r.key); }
      row.addEventListener("click", pick);
      row.addEventListener("keydown", function (e) { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); pick(); } });
      return row;
    });
    if (axisEl) {
      clear(axisEl); h("div", {}, axisEl); var ax = h("div", {}, axisEl);
      var marks = [t0].concat(dayStarts(t0 + 6 * 3600e3, t1 - 6 * 3600e3)).concat([t1]);
      var step = marks.length > 6 ? 2 : 1;
      marks.forEach(function (t, i) { if (i === 0 || i === marks.length - 1) h("span", { text: i === 0 ? fmt.when(t) : fmt.when(t) }, ax); else if (i % step === 0) h("span", { text: fmt.weekday(t) }, ax); });
    }
    var api = { select: function (key) { rows.forEach(function (r) { r.setAttribute("aria-current", String(r.getAttribute("data-k") === key)); }); } };
    api.select(spec.selected);
    return api;
  }

  /* ---------- answer cards ---------- */
  /*
   * spec: { options:[{key,title,pill:{text,tone},metric,bullets[],recommended}], current,
   *   onPreview(key), answerKey ("answer"), statusEl }
   * "Choose this" calls hub.prefill(answerKey, key) and hub.answer.set(answerKey, label, key).
   */
  function answerCards(el, spec) {
    clear(el);
    var qid = spec.answerKey || "answer", chosen = null;
    var cards = (spec.options || []).map(function (o) {
      var c = h("article", { class: "kc-answer", "data-k": o.key, tabindex: "0" }, el);
      var head = h("div", { class: "kc-answer__head" }, c);
      h("h3", { text: o.title }, head);
      if (o.pill && o.pill.text) head.appendChild(pill(o.pill.text, o.pill.tone));
      if (o.metric) h("div", { class: "kc-answer__metric", text: o.metric }, c);
      if (o.bullets && o.bullets.length) { var ul = h("ul", {}, c); o.bullets.forEach(function (b) { h("li", { text: b }, ul); }); }
      var btn = h("button", { type: "button", class: "kc-btn" + (o.recommended ? " is-primary" : ""), text: "Choose this" }, c);
      c.addEventListener("click", function (e) {
        if (spec.onPreview) spec.onPreview(o.key);
        if (e.target.closest("button")) choose(o);
      });
      c.addEventListener("keydown", function (e) { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); if (spec.onPreview) spec.onPreview(o.key); } });
      return { el: c, btn: btn, o: o };
    });
    function choose(o) {
      chosen = o.key;
      var prefilled = answer.prefill(qid, o.key);
      var recorded = answer.set(qid, o.title, o.key);
      cards.forEach(function (c) { c.btn.textContent = c.o.key === chosen ? "Chosen" : "Choose this"; });
      if (spec.statusEl) spec.statusEl.textContent = recorded || prefilled ? "Your answer is now “" + o.title + "”. Send it from the Hub." : "Outside Notification Hub nothing is sent; this shows what “" + o.title + "” does.";
      if (spec.onChoose) spec.onChoose(o.key);
    }
    var api = {
      select: function (key) { cards.forEach(function (c) { c.el.setAttribute("aria-current", String(c.o.key === key)); }); },
      chosen: function () { return chosen; }
    };
    api.select(spec.current);
    return api;
  }

  /* ---------- recurrence ---------- */
  /* spec: the Hub's hub.context.recurrence shape, which producers copy:
     { count (asks so far, this one included), dates:[ISO, oldest first, this ask last],
       previousAnswers:[{answer, note, answeredAt}] (answered earlier asks), stop (sentence), labelFor(answerKey) } */
  function recurrence(el, spec) {
    clear(el);
    spec = spec || {};
    var all = (spec.dates || []).slice(), total = Math.max(num(spec.count) ? spec.count : 0, all.length, 1);
    var earlier = all.length >= total ? all.slice(0, total - 1) : all;
    var label = typeof spec.labelFor === "function" ? spec.labelFor : function (k) { return k; };
    var answers = (spec.previousAnswers || []).map(function (a) {
      if (typeof a === "string") return { label: a, at: null };
      return { label: label(a.answer) || a.answer || "", at: a.answeredAt || a.answered_at || null };
    }).filter(function (a) { return a.label; });
    // Put each answer under the latest earlier ask sent before it was given.
    var onDate = {};
    answers.forEach(function (a, i) {
      var at = a.at ? ms(a.at) : NaN, k = -1;
      for (var j = 0; j < earlier.length; j++) if (!isFinite(at) || ms(earlier[j]) <= at) k = j;
      if (!isFinite(at)) k = Math.min(i, earlier.length - 1);
      if (k >= 0) onDate[k] = a.label;
    });
    var box = h("div", { class: "kc-recur" }, el);
    h("strong", { class: "kc-recur__count", text: total + "×" }, box);
    var since = earlier.length ? " since " + fmt.date(earlier[0]) : "";
    var same = answers.length && answers.every(function (a) { return a.label === answers[0].label; });
    h("p", { html: "This is the <b>" + esc(ordinal(total)) + " time</b> you have been asked this" + esc(since) + "." + (same ? " " + (answers.length === 1 ? "Last time you answered" : "Every earlier answer was") + " “" + esc(answers[0].label) + "”." : "") }, box);
    if (earlier.length) {
      var track = h("div", { class: "kc-recur__track", role: "img", "aria-label": "Times this question was asked" }, box);
      earlier.slice(-6).forEach(function (d, i, arr) {
        var idx = earlier.length - arr.length + i;
        var dd = h("div", { class: "kc-recur__d" }, track);
        h("i", {}, dd);
        h("span", { html: "<b>" + esc(fmt.day(d)) + "</b>" + (onDate[idx] ? "<br>" + esc(onDate[idx]) : "<br><span class=\"kc-muted\">no answer</span>") }, dd);
      });
      var nowD = h("div", { class: "kc-recur__d is-now" }, track); h("i", {}, nowD); h("span", { html: "<b>Today</b>" }, nowD);
    }
    if (spec.stop) h("p", { class: "kc-recur__stop", text: spec.stop }, box);
    return el;
  }
  function ordinal(n) { var w = ["zeroth", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth"]; return n < w.length ? w[n] : n + "th"; }

  /* ---------- dumbbell: planned vs actual ---------- */
  /* spec: { rows:[{key,label,planned,actual,hours,tone}], totalHours } shares are 0..1 */
  function dumbbell(el, spec) {
    var rows = spec.rows || [];
    var total = num(spec.totalHours) ? spec.totalHours : rows.reduce(function (a, r) { return a + (r.hours || 0); }, 0);
    var top = Math.min(1, Math.ceil(Math.max.apply(null, [0.2].concat(rows.map(function (r) { return Math.max(r.planned || 0, r.actual || 0) * 1.06; }))) * 10) / 10);
    onWidth(el, function (W) {
      var narrow = W < 520, lab = narrow ? 138 : 160, rowH = 34, m = { l: lab, r: 46, t: 12, b: 24 }, Ht = m.t + m.b + rowH * rows.length;
      var x = function (v) { return m.l + (v / top) * (W - m.l - m.r); };
      var svg = svgRoot(el, W, Ht);
      var ticks = [0, top / 2, top];
      ticks.forEach(function (g) { s("line", { x1: x(g), x2: x(g), y1: m.t - 6, y2: Ht - m.b, class: "kc-grid" }, svg); s("text", { x: x(g), y: Ht - 6, "text-anchor": "middle", text: Math.round(g * 100) + "%" }, svg); });
      rows.forEach(function (r, i) {
        var cy = m.t + rowH * i + rowH / 2, over = r.actual > r.planned, g = s("g", { tabindex: 0 }, svg);
        s("rect", { x: 0, y: cy - rowH / 2, width: W, height: rowH, fill: "transparent", class: "kc-focus" }, g);
        s("rect", { x: 0, y: cy - 5, width: 10, height: 10, rx: 3, fill: tone(r.tone) }, g);
        s("text", { x: 16, y: cy + 4, class: "kc-t-ink kc-t-body", text: truncate(r.label, Math.floor((lab - 22) / 6.6)) }, g);
        if (num(r.planned) && num(r.actual)) s("line", { x1: x(Math.min(r.planned, r.actual)), x2: x(Math.max(r.planned, r.actual)), y1: cy, y2: cy, stroke: over ? "var(--kc-tone-2)" : "var(--kc-tone-1)", "stroke-width": 4, "stroke-linecap": "round" }, g);
        if (num(r.planned)) s("circle", { cx: x(r.planned), cy: cy, r: 6, fill: "var(--hub-surface)", stroke: "var(--hub-muted)", "stroke-width": 2 }, g);
        if (num(r.actual)) s("circle", { cx: x(r.actual), cy: cy, r: 6.5, fill: "var(--hub-fg)", stroke: "var(--hub-surface)", "stroke-width": 2 }, g);
        if (num(r.hours)) s("text", { x: W - 2, y: cy + 4, "text-anchor": "end", class: "kc-t-ink", text: Math.round(r.hours) + " h" }, g);
        bindTip(g, function () {
          var diff = num(r.actual) && num(r.planned) ? Math.round(Math.abs(r.actual - r.planned) * total) : null;
          return "<b>" + esc(r.label) + "</b><br><span class=\"kc-num\">" + (num(r.hours) ? Math.round(r.hours) + " h · " : "") + fmt.share(r.actual) + " actual · " + fmt.share(r.planned) + " planned</span>" + (diff !== null && total ? "<br>" + (over ? "Over" : "Under") + " plan by " + diff + " h" : "");
        });
      });
    });
  }

  /* ---------- funnel ---------- */
  /* spec: { steps:[{label,value,note,tone}] } */
  function funnel(el, spec) {
    clear(el); el.classList.add("kc-fun");
    var steps = spec.steps || [], top = steps.length ? Math.max.apply(null, steps.map(function (st) { return st.value || 0; })) : 1;
    steps.forEach(function (st) {
      var row = h("div", { class: "kc-fun-row" }, el);
      var l = h("div", { class: "kc-fun-l" }, row);
      h("b", { text: st.label }, l);
      if (st.note) h("span", { text: st.note }, l);
      var bar = h("div", { class: "kc-fun-bar", role: "img", "aria-label": st.label + ": " + st.value }, row);
      var i = h("i", {}, bar);
      i.style.width = Math.max(top ? (st.value / top) * 100 : 0, 0.8) + "%";
      i.style.background = tone(st.tone || "ink");
      h("div", { class: "kc-fun-n", text: fmt.n(st.value) }, row);
    });
  }

  /* ---------- proportion bar (+ legend, + groups) ---------- */
  /* spec: { parts:[{key,label,value,tone,why,items[]}], unit, thin, labelMin (share), legendEl, groupsEl } */
  function proportion(el, spec) {
    clear(el); el.classList.add("kc-prop"); if (spec.thin) el.classList.add("is-thin");
    var parts = (spec.parts || []).filter(function (p) { return p.value > 0; });
    var total = parts.reduce(function (a, p) { return a + p.value; }, 0) || 1, unit = spec.unit || "";
    el.setAttribute("role", "img");
    el.setAttribute("aria-label", parts.map(function (p) { return p.label + " " + fmt.n(p.value) + (unit ? " " + unit : ""); }).join(", "));
    parts.forEach(function (p) {
      var share = p.value / total;
      var i = h("i", { text: !spec.thin && share >= (spec.labelMin || 0.08) ? fmt.n(p.value) + (unit && unit.length <= 2 ? " " + unit : "") : "" }, el);
      i.style.flex = String(p.value); i.style.background = tone(p.tone);
      bindTip(i, "<b>" + esc(p.label) + "</b><br><span class=\"kc-num\">" + esc(fmt.n(p.value)) + (unit ? " " + esc(unit) : "") + " · " + Math.round(share * 100) + "%</span>");
    });
    if (spec.legendEl) legend(spec.legendEl, parts.map(function (p) { return { label: p.label, color: tone(p.tone), value: fmt.n(p.value) + (unit && unit.length <= 2 ? " " + unit : "") }; }));
    if (spec.groupsEl) {
      clear(spec.groupsEl); spec.groupsEl.classList.add("kc-groups");
      parts.forEach(function (p) {
        var d = h("details", { class: "kc-group" }, spec.groupsEl);
        var sm = h("summary", {}, d);
        var sw = h("i", { class: "kc-sw" }, sm); sw.style.background = tone(p.tone);
        h("span", { text: p.label }, sm);
        h("span", { class: "kc-num", text: fmt.n(p.value) }, sm);
        if (p.why) h("span", { class: "kc-group__why", text: p.why }, sm);
        if (p.items && p.items.length) { var ul = h("ul", {}, d); p.items.forEach(function (it) { h("li", { text: it }, ul); }); }
      });
    }
  }
  /* rows: [{label, parts:[{label,value,tone}], value (text at right)}] */
  function shareBars(el, rows) {
    clear(el); el.classList.add("kc-share");
    rows.forEach(function (r) {
      h("span", { text: r.label }, el);
      var bar = h("div", {}, el);
      proportion(bar, { parts: r.parts, thin: true, unit: r.unit || "h" });
      h("span", { class: "kc-num", text: r.value }, el);
    });
  }

  /* ---------- sparkline ---------- */
  /* spec: { values:[n], tone, max, label, height } */
  function sparkline(el, spec) {
    var vals = spec.values || [];
    onWidth(el, function (W) {
      var Hh = spec.height || 36, n = vals.length, mx = Math.max(spec.max || 0, Math.max.apply(null, vals.concat([1])));
      var x = function (i) { return n > 1 ? 2 + (i / (n - 1)) * (W - 6) : W / 2; };
      var y = function (v) { return Hh - 3 - (v / mx) * (Hh - 8); };
      var svg = svgRoot(el, W, Hh, spec.label);
      if (!n) return;
      var pts = vals.map(function (v, i) { return x(i) + "," + y(v); }).join(" ");
      s("polygon", { points: x(0) + "," + Hh + " " + pts + " " + x(n - 1) + "," + Hh, fill: tone(spec.tone || 2), "fill-opacity": .16 }, svg);
      s("polyline", { points: pts, fill: "none", stroke: tone(spec.tone || 2), "stroke-width": 1.75, "stroke-linejoin": "round" }, svg);
      s("circle", { cx: x(n - 1), cy: y(vals[n - 1]), r: 3, fill: tone(spec.tone || 2) }, svg);
    });
  }

  /* ---------- stacked daily bars ---------- */
  /* spec: { days:[{t, values:{key:n}}], series:[{key,label,tone}] } */
  function stackedBars(el, spec) {
    var days = spec.days || [], series = spec.series || [];
    var top = niceMax(Math.max.apply(null, [1].concat(days.map(function (d) { return series.reduce(function (a, sr) { return a + ((d.values || {})[sr.key] || 0); }, 0); }))));
    onWidth(el, function (W) {
      var Ht = 184, m = { l: 28, r: 6, t: 8, b: 24 }, n = days.length || 1, bw = (W - m.l - m.r) / n;
      var y = function (v) { return m.t + (1 - v / top) * (Ht - m.t - m.b); };
      var svg = svgRoot(el, W, Ht);
      [0, top / 2, top].forEach(function (g) { s("line", { x1: m.l, x2: W - m.r, y1: y(g), y2: y(g), class: "kc-grid" }, svg); s("text", { x: m.l - 6, y: y(g) + 4, "text-anchor": "end", text: fmt.n(g) }, svg); });
      var every = Math.max(1, Math.ceil(56 / bw));
      days.forEach(function (d, i) {
        var acc = 0, x0 = m.l + i * bw + 2, w = Math.max(3, bw - 4), g = s("g", { tabindex: 0 }, svg);
        series.forEach(function (sr) {
          var v = (d.values || {})[sr.key] || 0; if (!v) return;
          s("rect", { x: x0, y: y(acc + v) + 1, width: w, height: Math.max(1, y(acc) - y(acc + v) - 2), rx: 2, fill: tone(sr.tone) }, g); acc += v;
        });
        s("rect", { x: x0, y: m.t, width: w, height: Ht - m.t - m.b, fill: "transparent", class: "kc-focus" }, g);
        if ((n - 1 - i) % every === 0) s("text", { x: x0 + w / 2, y: Ht - 6, "text-anchor": "middle", text: fmt.date(d.t) }, svg);
        bindTip(g, function () {
          var lines = series.map(function (sr) { var v = (d.values || {})[sr.key] || 0; return v ? esc(sr.label) + ": " + v : ""; }).filter(Boolean);
          return "<b>" + esc(fmt.dayDate(d.t)) + "</b><br>" + (lines.length ? lines.join("<br>") : "None");
        });
      });
    });
  }

  /* ---------- target chart: money earned toward a dated target ---------- */
  /* spec: { start, end (deadline), now, target, earned:[{t,amount}], currency, pendingLabel } */
  function targetChart(el, spec) {
    onWidth(el, function (W) {
      var Ht = W < 520 ? 172 : 196, m = { l: 50, r: 14, t: 20, b: 26 }, t0 = ms(spec.start), t1 = ms(spec.end), now = ms(spec.now), top = spec.target * 1.14;
      var x = function (t) { return m.l + ((ms(t) - t0) / (t1 - t0)) * (W - m.l - m.r); };
      var y = function (v) { return m.t + (1 - v / top) * (Ht - m.t - m.b); };
      var svg = svgRoot(el, W, Ht);
      var earned = spec.earned || [], last = earned.length ? earned[earned.length - 1] : { t: now, amount: 0 };
      if (now < t1) s("rect", { x: x(now), y: m.t, width: x(t1) - x(now), height: Ht - m.t - m.b, fill: "var(--hub-surface-muted)" }, svg);
      [0, spec.target / 2, spec.target].forEach(function (v) { s("line", { x1: m.l, x2: W - m.r, y1: y(v), y2: y(v), class: "kc-grid" }, svg); s("text", { x: m.l - 6, y: y(v) + 4, "text-anchor": "end", text: fmt.money(v, spec.currency) }, svg); });
      var span = (t1 - t0) / 864e5, step = span > 60 ? 14 : 7, lastX = -1e9;
      for (var d = t0; d <= t1; d += step * 864e5) { if (x(d) - lastX > 54) { s("text", { x: x(d), y: Ht - 8, "text-anchor": "middle", text: fmt.date(d) }, svg); lastX = x(d); } }
      if (now < t1) s("line", { x1: x(now), y1: y(last.amount), x2: x(t1), y2: y(spec.target), stroke: "var(--hub-muted)", "stroke-width": 1.75, "stroke-dasharray": "5 5" }, svg);
      var pts = earned.map(function (p) { return x(p.t) + "," + y(p.amount); });
      if (earned.length && ms(last.t) < now) pts.push(x(now) + "," + y(last.amount));
      if (pts.length > 1) s("polyline", { points: pts.join(" "), fill: "none", stroke: "var(--hub-fg)", "stroke-width": 3, "stroke-linecap": "round", "stroke-linejoin": "round" }, svg);
      s("circle", { cx: x(now), cy: y(last.amount), r: 6, fill: "var(--hub-fg)", stroke: "var(--hub-surface)", "stroke-width": 2 }, svg);
      s("text", { x: x(now), y: m.t - 6, "text-anchor": "middle", class: "kc-t-strong", text: "Today" }, svg);
      s("circle", { cx: x(t1), cy: y(spec.target), r: 6, fill: "var(--hub-surface)", stroke: "var(--hub-fg)", "stroke-width": 2 }, svg);
      s("text", { x: x(t1) - 10, y: y(spec.target) - 10, "text-anchor": "end", class: "kc-t-strong kc-halo", text: fmt.money(spec.target, spec.currency) + " · " + fmt.date(t1) }, svg);
      var note = fmt.money(last.amount, spec.currency) + " confirmed" + (spec.pendingLabel ? " · " + spec.pendingLabel : "");
      var nx = x(now), anchor = nx > W * 0.6 ? "end" : "start";
      fitText(s("text", { x: anchor === "end" ? nx - 10 : nx + 10, y: y(last.amount) - 10, "text-anchor": anchor, class: "kc-t-ink kc-halo", text: note }, svg), m.l + 4, W - 4);
      if (now < t1) {
        var hit = s("rect", { x: x(now), y: m.t, width: x(t1) - x(now), height: Ht - m.t - m.b, fill: "transparent" }, svg);
        var daysLeft = Math.max(0, Math.ceil((t1 - now) / 864e5)), need = daysLeft ? (spec.target - last.amount) / daysLeft : 0;
        bindTip(hit, "<b>" + daysLeft + " " + plural(daysLeft, "day") + " left</b><br>Needs about " + esc(fmt.money(need, spec.currency)) + " a day from today.");
      }
    });
  }

  /* ---------- slot schedule (Gantt) ---------- */
  /*
   * spec: { start, hours, slots, baseSlots, blocks:[{slot,from,to,lane,label,running,changed}],
   *   lanes:{key:{label,tone}}, now }
   */
  function slotSchedule(el) {
    var spec = null;
    var sizer = onWidth(el, function (W) { if (spec) draw(W); });
    function draw(W) {
      var narrow = W < 560, slots = Math.max(1, spec.slots || 1), hrs = spec.hours || 24;
      var halves = narrow && hrs > 12 ? 2 : 1, span = hrs / halves;
      var m = { l: narrow ? 28 : 58, r: 6, t: 6, b: 22 }, rowH = narrow ? 44 : 40, gap = 18;
      var bandH = rowH * slots + m.b, Ht = m.t + halves * bandH + (halves - 1) * gap;
      var t0 = ms(spec.start), lanes = spec.lanes || {}, now = ms(spec.now);
      var svg = svgRoot(el, W, Ht);
      for (var k = 0; k < halves; k++) {
        var a0 = t0 + k * span * 3600e3, a1 = a0 + span * 3600e3, top = m.t + k * (bandH + gap);
        var x = (function (a0, a1) { return function (t) { return m.l + ((ms(t) - a0) / (a1 - a0)) * (W - m.l - m.r); }; })(a0, a1);
        for (var r = 0; r < slots; r++) {
          var extra = spec.baseSlots && r >= spec.baseSlots;
          if (extra) s("rect", { x: m.l, y: top + rowH * r + 2, width: W - m.l - m.r, height: rowH - 4, rx: 6, fill: "var(--kc-accent-soft)", "fill-opacity": .6 }, svg);
          s("text", { x: 0, y: top + rowH * r + rowH / 2 + 4, class: extra ? "kc-t-strong" : null, style: extra ? "fill:var(--kc-accent)" : null, text: narrow ? "S" + (r + 1) : "Slot " + (r + 1) }, svg);
        }
        for (var hh = 0; hh <= span; hh++) {
          var t = a0 + hh * 3600e3, edge = hh === 0 || hh === span;
          if (!edge && hourOf(t) % 3 !== 0) continue;
          var tx = x(t);
          s("line", { x1: tx, x2: tx, y1: top, y2: top + rowH * slots, class: "kc-grid" }, svg);
          if (edge || (tx - m.l > 40 && W - m.r - tx > 40)) s("text", { x: tx, y: top + rowH * slots + 15, "text-anchor": hh === 0 ? "start" : hh === span ? "end" : "middle", text: fmt.hm(t) }, svg);
        }
        (spec.blocks || []).forEach(function (b) {
          if (!(b.slot >= 1 && b.slot <= slots)) return;
          var bf = ms(b.from), bt = ms(b.to), a = Math.max(a0, bf), z = Math.min(a1, bt);
          if (!(z > a)) return;
          var lane = lanes[b.lane] || { label: b.lane, tone: 7 };
          var bx = x(a) + 1, bw = Math.max(2, x(z) - x(a) - 2), by = top + rowH * (b.slot - 1) + 4, bh = rowH - 8, g = s("g", { tabindex: 0 }, svg);
          s("rect", { x: bx, y: by, width: bw, height: bh, rx: 6, fill: tone(lane.tone), "fill-opacity": .2, stroke: b.changed ? "var(--hub-fg)" : "none", "stroke-width": b.changed ? 2 : 0, class: "kc-focus" }, g);
          s("rect", { x: bx, y: by, width: 4, height: bh, rx: 2, fill: tone(lane.tone) }, g);
          var tag = b.running && bw > 64;
          if (tag) s("text", { x: bx + 10, y: by + 12, style: "font-size:9px;letter-spacing:.06em", text: "RUNNING" }, g);
          var maxc = Math.floor((bw - 14) / 6.3);
          if (maxc >= 3) s("text", { x: bx + 10, y: by + bh / 2 + (tag ? 9 : 4), class: "kc-t-ink kc-t-body", style: "font-size:11.5px", text: truncate(b.label, maxc) }, g);
          bindTip(g, "<b>" + esc(b.label) + "</b><br><span class=\"kc-num\">" + esc(fmt.hm(bf)) + " \u2192 " + esc(fmt.hm(bt)) + " \u00b7 " + esc(lane.label || b.lane) + "</span>" + (b.running ? "<br>Already running" : "") + (b.changed ? "<br>Changed by this choice" : ""));
        });
        if (now > a0 && now < a1) s("line", { x1: x(now), x2: x(now), y1: top - 2, y2: top + rowH * slots, stroke: "var(--hub-fg)", "stroke-width": 1, "stroke-dasharray": "2 3" }, svg);
      }
    }
    return { update: function (next) { spec = next; sizer.redraw(); } };
  }
  /** Hours per lane from schedule blocks, clipped to the window. */
  function laneHours(spec) {
    var out = {}, all = 0, t0 = ms(spec.start), t1 = t0 + (spec.hours || 24) * 3600e3;
    (spec.blocks || []).forEach(function (b) {
      if (!(b.slot >= 1 && b.slot <= (spec.slots || 1))) return;
      var hrs = Math.max(0, Math.min(t1, ms(b.to)) - Math.max(t0, ms(b.from))) / 3600e3;
      out[b.lane] = (out[b.lane] || 0) + hrs; all += hrs;
    });
    return { byLane: out, total: all };
  }

  /* ---------- check map: hours lost against harm ---------- */
  /* spec: { items:[{id,label,hours,harm,on,locked,lockText}], zone:{minHours,maxHarm,label}, onToggle(id), harmTop, harmBottom }
   locked items are drawn dashed and cannot be toggled (hard stops, or checks with no guard mapping). */
  function checkMap(el, spec) {
    var api = {};
    var sizer = onWidth(el, function (W) {
      var items = spec.items || [];
      var Ht = W < 520 ? 276 : 316, m = { l: 18, r: 16, t: 24, b: 42 };
      var lo = Math.log10(Math.max(0.1, Math.min.apply(null, items.map(function (c) { return Math.max(0.1, c.hours); }).concat([0.5])) * 0.6));
      var hi = Math.log10(Math.max.apply(null, items.map(function (c) { return c.hours; }).concat([2])) * 1.6);
      var x = function (hrs) { return m.l + ((Math.log10(Math.max(0.1, hrs)) - lo) / (hi - lo)) * (W - m.l - m.r); };
      var y = function (r) { return m.t + (clamp(r, 0, 10) / 10) * (Ht - m.t - m.b); };
      var svg = svgRoot(el, W, Ht);
      var z = spec.zone;
      if (z) {
        s("rect", { x: x(z.minHours), y: m.t, width: W - m.r - x(z.minHours), height: y(z.maxHarm) - m.t, rx: 10, fill: "var(--kc-tone-2)", "fill-opacity": .09 }, svg);
        if (W - m.r - x(z.minHours) > 120) s("text", { x: W - m.r - 8, y: y(z.maxHarm) - 8, "text-anchor": "end", class: "kc-t-ink", text: z.label || "Costs time, guards little" }, svg);
      }
      s("line", { x1: m.l, x2: W - m.r, y1: Ht - m.b, y2: Ht - m.b, class: "kc-grid" }, svg);
      s("line", { x1: m.l, x2: m.l, y1: m.t, y2: Ht - m.b, class: "kc-grid" }, svg);
      [0.5, 1, 2, 5, 10, 20, 50, 100].forEach(function (t) { if (Math.log10(t) > lo && Math.log10(t) < hi) s("text", { x: x(t), y: Ht - m.b + 15, "text-anchor": "middle", text: t + " h" }, svg); });
      s("text", { x: W - m.r, y: Ht - 4, "text-anchor": "end", text: "hours lost per week →" }, svg);
      s("text", { x: m.l + 4, y: m.t - 8, text: spec.harmTop || "↑ git restores it" }, svg);
      s("text", { x: m.l, y: Ht - 4, text: spec.harmBottom || "↓ can't be undone" }, svg);
      items.forEach(function (c) {
        var cx = x(c.hours), cy = y(c.harm), lit = !!c.on;
        var g = c.locked ? s("g", { "aria-label": c.label + ": " + (c.lockText || "always kept") }, svg) : s("g", { tabindex: 0, role: "button", "aria-pressed": String(lit), "aria-label": c.label }, svg);
        s("circle", { cx: cx, cy: cy, r: 18, fill: "transparent", class: "kc-focus" }, g);
        s("circle", { cx: cx, cy: cy, r: 11, fill: lit ? "var(--kc-tone-2)" : "var(--hub-surface)", stroke: lit ? "var(--hub-surface)" : "var(--hub-muted)", "stroke-width": 2, "stroke-dasharray": c.locked ? "3 2.5" : null }, g);
        s("text", { x: cx, y: cy + 4, "text-anchor": "middle", style: "font-weight:700;fill:" + (lit ? "var(--hub-surface)" : "var(--hub-fg)"), text: String(c.id) }, g);
        bindTip(g, function () { return "<b>" + esc(c.label) + "</b><br><span class=\"kc-num\">" + esc(fmt.n(c.hours, c.hours < 10 ? 1 : 0)) + " h/week · harm " + esc(fmt.n(c.harm, 1)) + "/10</span><br>" + (c.locked ? esc(c.lockText || "Always kept") : c.on ? "Loosen" : "Keep"); });
        if (c.locked) return;
        function toggle() { if (spec.onToggle) spec.onToggle(c.id); }
        g.addEventListener("click", toggle);
        g.addEventListener("keydown", function (e) { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); toggle(); } });
      });
    });
    api.redraw = function () { sizer.redraw(); };
    api.update = function (next) { spec = next; sizer.redraw(); };
    return api;
  }

  /* ---------- slip chart: a gate forecast at each review ---------- */
  /* spec: { reviews:[{label,from,to}], proposed:{label,from,to}|null, refs:[{t,label,tone}] } */
  function slipChart(el, spec) {
    onWidth(el, function (W) {
      var pts = (spec.reviews || []).map(function (r) { return { label: r.label, lo: ms(r.from), hi: ms(r.to), mid: (ms(r.from) + ms(r.to)) / 2 }; });
      var you = spec.proposed ? { label: spec.proposed.label || "Your weights", lo: ms(spec.proposed.from), hi: ms(spec.proposed.to), mid: (ms(spec.proposed.from) + ms(spec.proposed.to)) / 2 } : null;
      var all = you ? pts.concat([you]) : pts, n = Math.max(1, all.length - 1);
      if (!all.length) { clear(el); return; }
      var pad = 4 * 864e5, lo = Math.min.apply(null, all.map(function (p) { return p.lo; })) - pad, hi = Math.max.apply(null, all.map(function (p) { return p.hi; })) + pad;
      var Ht = 164, m = { l: 54, r: 12, t: 12, b: 24 };
      var x = function (i) { return all.length === 1 ? (m.l + W - m.r) / 2 : m.l + (i / n) * (W - m.l - m.r); };
      var y = function (d) { return m.t + (1 - (d - lo) / (hi - lo)) * (Ht - m.t - m.b); };
      var svg = svgRoot(el, W, Ht);
      [lo + pad, (lo + hi) / 2, hi - pad].forEach(function (d) { s("line", { x1: m.l, x2: W - m.r, y1: y(d), y2: y(d), class: "kc-grid" }, svg); s("text", { x: m.l - 6, y: y(d) + 4, "text-anchor": "end", text: fmt.date(d) }, svg); });
      (spec.refs || []).forEach(function (r) { var d = ms(r.t); if (d > lo && d < hi) { s("line", { x1: m.l, x2: W - m.r, y1: y(d), y2: y(d), stroke: tone(r.tone || "crit"), "stroke-dasharray": "4 3" }, svg); s("text", { x: W - m.r, y: y(d) - 4, "text-anchor": "end", style: "fill:" + tone(r.tone || "crit"), text: r.label }, svg); } });
      if (pts.length > 1) s("polygon", { points: pts.map(function (p, i) { return x(i) + "," + y(p.hi); }).concat(pts.slice().reverse().map(function (p, i) { return x(pts.length - 1 - i) + "," + y(p.lo); })).join(" "), fill: "var(--kc-tone-3)", "fill-opacity": .25 }, svg);
      else if (pts.length === 1) s("line", { x1: x(0), x2: x(0), y1: y(pts[0].lo), y2: y(pts[0].hi), stroke: "var(--kc-tone-3)", "stroke-width": 8, "stroke-linecap": "round", "stroke-opacity": .35 }, svg);
      if (pts.length > 1) s("polyline", { points: pts.map(function (p, i) { return x(i) + "," + y(p.mid); }).join(" "), fill: "none", stroke: "var(--hub-fg)", "stroke-width": 2.25 }, svg);
      if (you && pts.length) {
        var li = pts.length - 1;
        s("line", { x1: x(li), y1: y(pts[li].mid), x2: x(li + 1), y2: y(you.mid), stroke: "var(--kc-accent)", "stroke-width": 2.25, "stroke-dasharray": "5 4" }, svg);
        s("line", { x1: x(li + 1), x2: x(li + 1), y1: y(you.lo), y2: y(you.hi), stroke: "var(--kc-accent)", "stroke-width": 6, "stroke-linecap": "round", "stroke-opacity": .35 }, svg);
      }
      all.forEach(function (p, i) {
        var isY = p === you, isLast = !isY && i === pts.length - 1;
        var g = s("g", {}, svg);
        s("circle", { cx: x(i), cy: y(p.mid), r: 5.5, fill: isY ? "var(--kc-accent)" : isLast ? "var(--hub-fg)" : "var(--hub-surface)", stroke: isY ? "var(--kc-accent)" : "var(--hub-fg)", "stroke-width": 2 }, g);
        s("text", { x: x(i), y: Ht - 6, "text-anchor": all.length === 1 ? "middle" : i === 0 ? "start" : i === all.length - 1 ? "end" : "middle", style: isY ? "fill:var(--kc-accent);font-weight:600" : null, text: p.label }, svg);
        var hit = s("circle", { cx: x(i), cy: y(p.mid), r: 16, fill: "transparent" }, g);
        bindTip(hit, "<b>" + esc(p.label) + "</b><br><span class=\"kc-num\">" + esc(fmt.date(p.lo)) + " – " + esc(fmt.date(p.hi)) + "</span>");
      });
    });
  }

  /* ---------- roadmap timeline marks ---------- */
  function hatchDefs(svg, id) {
    var defs = s("defs", {}, svg), p = s("pattern", { id: id, width: 6, height: 6, patternUnits: "userSpaceOnUse", patternTransform: "rotate(45)" }, defs);
    s("rect", { width: 6, height: 6, fill: "var(--hub-surface)" }, p);
    s("rect", { width: 3, height: 6, fill: "var(--kc-warn-soft)" }, p);
  }
  /*
   * One roadmap row's marks. row: { phases:[{from,to,state}], gate:{from,to}|null, gateBefore:{from,to}|null,
   * noPlanFrom (ms)|null, paused } ; x maps ms -> px ; end (ms)
   */
  function roadmapMarks(g, row, x, cy, hh, end, hatchId) {
    var cap = function (t) { return Math.min(end, t); };
    if (row.gateBefore) s("rect", { x: x(row.gateBefore.from), y: cy - hh, width: Math.max(2, x(cap(row.gateBefore.to)) - x(row.gateBefore.from)), height: hh * 2, rx: 5, fill: "none", stroke: "var(--hub-muted)", "stroke-width": 1.5, "stroke-dasharray": "3 3" }, g);
    if (row.gate && row.gate.from < end) {
      s("rect", { x: x(row.gate.from), y: cy - hh, width: Math.max(2, x(cap(row.gate.to)) - x(row.gate.from)), height: hh * 2, rx: 5, fill: "var(--kc-tone-3)", "fill-opacity": .3, class: "kc-anim" }, g);
      var mid = Math.min(end - 864e5, (row.gate.from + row.gate.to) / 2);
      s("path", { d: "M" + x(mid) + "," + (cy - 6) + " l6,6 l-6,6 l-6,-6z", fill: "var(--kc-tone-3)", stroke: "var(--hub-surface)", "stroke-width": 1.5 }, g);
    }
    (row.phases || []).forEach(function (p) {
      if (p.from >= end) return;
      s("rect", { x: x(p.from) + 1, y: cy - 5, width: Math.max(3, x(cap(p.to)) - x(p.from) - 2), height: 10, rx: 4, fill: p.state === "done" ? "var(--hub-success)" : p.state === "active" ? "var(--kc-accent)" : "var(--kc-accent-soft)", stroke: p.state === "planned" ? "var(--kc-accent)" : "none" }, g);
    });
    if (row.noPlanFrom && row.noPlanFrom < end) s("rect", { x: x(row.noPlanFrom), y: cy - 8, width: x(end) - x(row.noPlanFrom), height: 16, rx: 5, fill: "url(#" + hatchId + ")", stroke: "var(--hub-warning)", "stroke-opacity": .6 }, g);
    if (row.paused && row.pausedAt) s("text", { x: x(row.pausedAt) + 8, y: cy + 4, class: "kc-t-ink", style: "font-size:10.5px", text: "Paused" }, g);
  }
  function refLines(svg, x, y0, y1, refs, labels) {
    (refs || []).forEach(function (r, i) {
      var c = tone(r.tone || "ink"), isNow = r.kind === "now";
      s("line", { x1: x(ms(r.t)), x2: x(ms(r.t)), y1: y0, y2: y1, stroke: c, "stroke-width": isNow ? 2 : 1.25, "stroke-dasharray": isNow ? null : "4 3" }, svg);
      if (labels) s("text", { x: x(ms(r.t)), y: y0 - 6, "text-anchor": "middle", class: "kc-t-strong", style: "fill:" + c, text: r.label }, svg);
    });
  }


  /* ---------- measured results: before/after bar pairs ---------- */
  /* items: [{title, unit ('x' = ratio), values:[{label,value}], better:'lower'|'higher', verdict:'good'|'warn'|'bad', note, evidence}] */
  var VERDICT = { good: ["Win", "good", "var(--hub-success)"], warn: ["Watch", "warn", "var(--hub-warning)"], bad: ["Problem", "crit", "var(--hub-danger)"] };
  function measureValue(v, unit) {
    if (!num(v)) return "–";
    var a = Math.abs(v), t = a === 0 ? "0" : a < 1 ? String(+v.toPrecision(3)) : a < 100 ? String(+v.toFixed(2)) : Math.round(v).toLocaleString(LOCALE);
    return unit === "x" ? t + "×" : t + (unit ? " " + unit : "");
  }
  function measureChange(m) {
    var vals = m.values || [];
    if (vals.length < 2) return "";
    var a = vals[0].value, b = vals[vals.length - 1].value;
    if (!num(a) || !num(b)) return "";
    if (a === b) return "No change";
    if (m.unit === "x") { var rx = Math.max(a, b) / Math.max(Math.min(a, b), 1e-9); return (rx >= 100 ? Math.round(rx).toLocaleString(LOCALE) : rx.toFixed(1)) + "× " + (b < a ? "less" : "more"); }
    var lo = Math.min(a, b), hi = Math.max(a, b);
    if (lo > 0 && hi / lo >= 1.5) { var f = hi / lo; return (f >= 10 ? Math.round(f).toLocaleString(LOCALE) : f.toFixed(1)) + "× " + (b < a ? "lower" : "higher"); }
    var pct = (b - a) / Math.abs(a || 1) * 100;
    return (pct > 0 ? "+" : "−") + Math.abs(pct).toFixed(Math.abs(pct) < 10 ? 1 : 0) + "%";
  }
  function measureCards(el, items) {
    clear(el); el.classList.add("kc-wins");
    (items || []).forEach(function (m) {
      var v = VERDICT[m.verdict] || VERDICT.warn, vals = m.values || [];
      var mx = Math.max.apply(null, [0].concat(vals.map(function (x) { return num(x.value) ? x.value : 0; }))) || 1;
      var c = h("article", { class: "kc-win" }, el);
      c.style.setProperty("--kc-sev", v[2]);
      var hd = h("div", { class: "kc-win__h" }, c);
      h("h3", { text: m.title }, hd);
      hd.appendChild(pill(v[0], v[1]));
      var bars = h("div", { class: "kc-pairs", role: "img", "aria-label": m.title + ": " + vals.map(function (x) { return x.label + " " + measureValue(x.value, m.unit); }).join(", ") }, c);
      vals.forEach(function (x, i) {
        var last = i === vals.length - 1, r = h("div", { class: "kc-pair" + (last ? " is-result" : "") }, bars);
        h("span", { class: "kc-pair__l", text: x.label }, r);
        var t = h("div", { class: "kc-pair__t" }, r), b = h("i", {}, t);
        b.style.width = Math.max(1.5, (num(x.value) ? x.value : 0) / mx * 100) + "%";
        if (last) b.style.background = v[2];
        h("span", { class: "kc-num kc-pair__v", text: measureValue(x.value, m.unit) }, r);
      });
      var ch = measureChange(m);
      if (ch) { var cl = h("p", { class: "kc-win__chg kc-num", text: ch }, c); cl.style.color = v[2]; }
      if (m.note) h("p", { class: "kc-win__note", text: m.note }, c);
      if (m.evidence) { var d = h("details", { class: "kc-win__ev" }, c); h("summary", { text: "Where the number comes from" }, d); h("p", { text: m.evidence }, d); }
    });
  }

  /* ---------- scorecard progress per dimension ---------- */
  /* spec: { dimensions:[{key,title,was?,now?,target?,level,reason,today}], scale:{max,unit}, levels:[names lowest first] }
     Numeric pairs show progress on a scale; dimensions without a recorded pair show the judged level and reason only. */
  function levelTone(level, levels) {
    var i = (levels || []).indexOf(level), n = (levels || []).length;
    if (i < 0 || n < 2) return "info";
    if (i === 0) return "";
    if (i === n - 1) return "good";
    return i / (n - 1) >= 0.5 ? "info" : "warn";
  }
  function progressRows(el, spec) {
    clear(el); el.classList.add("kc-progs");
    var max = spec.scale && num(spec.scale.max) ? spec.scale.max : 100, unit = spec.scale && spec.scale.unit !== undefined ? spec.scale.unit : "%";
    var pct = function (v) { return clamp(v / max * 100, 0, 100) + "%"; };
    var n0 = function (v) { return fmt.n(v, v % 1 ? 1 : 0); };
    var pts = unit === "%" ? " pts" : (unit ? " " + unit : "");
    var numeric = function (d) { return num(d.was) && num(d.now); };
    var hasNumeric = (spec.dimensions || []).some(numeric);
    if (hasNumeric) {
      var axis = h("div", { class: "kc-prog kc-prog--axis", "aria-hidden": "true" }, el);
      h("span", { class: "kc-prog__h kc-prog__cap", text: "Dimension and judged level" }, axis);
      var ax = h("div", { class: "kc-prog__scale" }, axis);
      [[0, "0"], [max / 2, n0(max / 2) + unit], [max, "Target " + n0(max) + unit]].forEach(function (t, k) { var s0 = h("span", { text: t[1] }, ax); s0.style.left = pct(t[0]); if (k === 2) s0.classList.add("is-end"); });
      h("span", { class: "kc-prog__v kc-prog__cap", text: "Was → now" }, axis);
    }
    (spec.dimensions || []).forEach(function (d) {
      var hasPair = numeric(d), target = num(d.target) ? d.target : max, gain = hasPair ? d.now - d.was : 0;
      var r = h("div", { class: "kc-prog" + (hasPair ? "" : " kc-prog--qual"), tabindex: "0", role: "button", "aria-expanded": "false" }, el);
      var hd = h("div", { class: "kc-prog__h" }, r);
      h("h3", { text: d.title }, hd);
      var right = h("span", { class: "kc-prog__r" }, hd);
      right.appendChild(pill(d.level, levelTone(d.level, spec.levels)));
      var v = h("span", { class: "kc-prog__v" + (hasPair ? " kc-num" : ""), text: hasPair ? "" : "Numeric progress unavailable" }, r);
      if (hasPair) {
        var dl = h("span", { class: "kc-num kc-prog__d" + (gain > 0 ? " is-up" : gain < 0 ? " is-down" : ""), text: gain > 0 ? "+" + n0(gain) + pts : gain < 0 ? "−" + n0(-gain) + pts : "no change" }, right);
        dl.title = gain > 0 ? "Gained this commission" : gain < 0 ? "Lost this commission" : "No change this commission";
        var tr = h("div", { class: "kc-track", role: "img", "aria-label": d.title + ": was " + n0(d.was) + unit + ", now " + n0(d.now) + unit + ", target " + n0(target) + unit }, r);
        var base = h("i", { class: "kc-track__was" }, tr); base.style.width = pct(Math.min(d.was, d.now));
        if (gain > 0) { var g = h("i", { class: "kc-track__gain" }, tr); g.style.left = pct(d.was); g.style.width = clamp(gain / max * 100, 0, 100) + "%"; }
        if (gain < 0) { var l = h("i", { class: "kc-track__loss" }, tr); l.style.left = pct(d.now); l.style.width = clamp(-gain / max * 100, 0, 100) + "%"; }
        var tg = h("b", { class: "kc-track__target" }, tr); tg.style.left = "calc(" + pct(target) + " - 1.5px)";
        if (gain !== 0) { h("span", { class: "kc-prog__was", text: n0(d.was) }, v); v.appendChild(document.createTextNode(" → ")); }
        h("b", { text: n0(d.now) + unit }, v);
        if (target !== max) h("span", { class: "kc-prog__was", text: " / " + n0(target) }, v);
      }
      var more = h("div", { class: "kc-prog__more", hidden: "" }, r);
      h("p", { html: "<b>Why “" + esc(d.level) + "”:</b> " + esc(d.reason) }, more);
      if (d.today) h("p", { html: "<b>Today:</b> " + esc(d.today) }, more);
      if (hasPair) h("p", { class: "kc-num", text: n0(Math.max(0, target - d.now)) + pts + " to the target" }, more);
      function flip() { var open = more.hidden; more.hidden = !open; r.setAttribute("aria-expanded", String(open)); }
      r.addEventListener("click", flip);
      r.addEventListener("keydown", function (e) { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); flip(); } });
    });
  }

  /* ---------- estimate ranges ---------- */
  /* spec: { items:[{id,title,low,high,sub,why}], unit, axisLabel, ref:{value,label} }
     Low-to-high range bars on one shared axis, with an optional reference marker (e.g. this commission's hours). */
  function rangeBars(el, spec) {
    clear(el); el.classList.add("kc-ranges");
    var items = spec.items || [], unit = spec.unit || "h", ref = spec.ref && num(spec.ref.value) ? spec.ref : null;
    var hiMax = Math.max.apply(null, [1, ref ? ref.value : 0].concat(items.map(function (x) { return num(x.high) ? x.high : 0; }))), step = niceMax(hiMax) / 10, top = Math.ceil(hiMax / step) * step;
    var p = function (v) { return clamp(v / top * 100, 0, 100) + "%"; };
    items.forEach(function (x, k) {
      var r = h("div", { class: "kc-range" }, el);
      var idc = h("span", { class: "kc-range__id kc-num" }, r);
      h("small", { text: String(k + 1) }, idc); idc.appendChild(document.createTextNode(x.id));
      var t = h("div", { class: "kc-range__t" }, r);
      h("span", { class: "kc-range__title", text: x.title }, t);
      if (x.why) h("span", { class: "kc-range__why", text: x.why }, t);
      if (x.sub) h("span", { class: "kc-range__sub", text: x.sub }, t);
      var bar = h("div", { class: "kc-range__bar", role: "img", "aria-label": x.title + ": " + fmt.n(x.low) + " to " + fmt.n(x.high) + " " + unit }, r);
      if (ref) { var rf = h("b", { class: "kc-range__ref" }, bar); rf.style.left = p(ref.value); }
      var i = h("i", {}, bar); i.style.left = p(x.low); i.style.width = "max(4px, " + clamp((x.high - x.low) / top * 100, 0, 100) + "%)";
      h("span", { class: "kc-num kc-range__v", text: fmt.n(x.low) + "–" + fmt.n(x.high) + " " + unit }, r);
      bindTip(bar, "<b>" + esc(x.id + " · " + x.title) + "</b><br><span class=\"kc-num\">" + esc(fmt.n(x.low) + "–" + fmt.n(x.high) + " " + unit) + "</span>" + (x.sub ? "<br>" + esc(x.sub) : ""));
    });
    var ax = h("div", { class: "kc-range kc-range--axis", "aria-hidden": "true" }, el);
    h("span", {}, ax); h("span", {}, ax);
    var ab = h("div", { class: "kc-range__axis" }, ax);
    [0, top / 2, top].forEach(function (v, k) { var s0 = h("span", { class: "kc-num", text: fmt.n(v) + (k === 2 ? " " + unit : "") }, ab); s0.style.left = p(v); });
    if (ref) { var rl = h("em", { class: "kc-range__reflabel", text: "▲ " + (ref.label || "") }, ab); rl.style.left = p(ref.value); if (ref.value / top < 0.22) rl.classList.add("is-left"); }
    h("span", { class: "kc-range__axlabel", text: spec.axisLabel || "" }, ax);
    return { top: top };
  }

  window.HubCharts = Object.freeze({
    version: "1.1.0",
    setZone: function (tz) { ZONE = tz || undefined; fmtCache = {}; },
    data: data, answer: answer, context: context, onContext: onContext, answerNote: answerNote,
    fmt: fmt, ms: ms, esc: esc, h: h, s: s, clear: clear, tone: tone, num: num, clamp: clamp, truncate: truncate, plural: plural, niceMax: niceMax, path: path, dayStarts: dayStarts,
    onWidth: onWidth, svgRoot: svgRoot, bindTip: bindTip, showTip: showTip, hideTip: hideTip,
    badge: badge, mountSource: mountSource, available: available, unavailable: unavailable, bindText: bindText,
    legend: legend, tiles: tiles, stats: stats, pill: pill, seg: seg,
    burnChart: burnChart, modeStrips: modeStrips, answerCards: answerCards, recurrence: recurrence,
    dumbbell: dumbbell, funnel: funnel, proportion: proportion, shareBars: shareBars, sparkline: sparkline,
    stackedBars: stackedBars, targetChart: targetChart, slotSchedule: slotSchedule, laneHours: laneHours,
    checkMap: checkMap, slipChart: slipChart, hatchDefs: hatchDefs, roadmapMarks: roadmapMarks, refLines: refLines,
    measureCards: measureCards, progressRows: progressRows, rangeBars: rangeBars, levelTone: levelTone
  });
})();
