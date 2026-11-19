import { node, slotNode, type Template } from "./spec";

/**
 * The template shelf. Every template is a real page composition made from RCL
 * assets (Button, Card, Input, Stat, Avatar, Chip, BoundedMeter) and semantic
 * layout nodes. The bar is a page that could ship: editorial type, deliberate
 * hierarchy, generous rhythm, and one clear action.
 */
export const TEMPLATES: Template[] = [
  {
    id: "aperture",
    name: "Aperture",
    tagline: "Analytics workspace",
    category: "analytics",
    idea: "A calm signal room: one number leads, the rest support it.",
    slotHint: "Primary action in the workspace header",
    display: "serif",
    radius: 16,
    palette: {
      bg: "#0c0e12",
      surface: "#141821",
      surfaceAlt: "#1b202b",
      ink: "#f4f6fb",
      muted: "#8b94a7",
      line: "#252c3a",
      accent: "#d6ff5e",
      accentInk: "#10140a",
      accentSoft: "rgba(214,255,94,.12)",
      secondary: "#8fb7ff",
    },
    spec: node("aperture.page", "page", { pad: 28 }, [
      node("aperture.nav", "row", { justify: "space-between", align: "center", gap: 18 }, [
        node("aperture.brand", "row", { gap: 9, align: "center" }, [
          node("aperture.mark", "swatch", { h: 22, w: 22, tone: "accent", radius: 7, shape: "orb" }),
          node("aperture.brandText", "text", { text: "Aperture", variant: "subtitle" }),
        ]),
        node("aperture.links", "row", { gap: 18, align: "center" }, [
          node("aperture.link1", "text", { text: "Overview", variant: "label", tone: "ink" }),
          node("aperture.link2", "text", { text: "Signals", variant: "label", tone: "muted" }),
          node("aperture.link3", "text", { text: "Reports", variant: "label", tone: "muted" }),
          node("aperture.avatar", "avatar", { name: "Ada Lovelace", size: "sm" }),
        ]),
      ]),
      node("aperture.rule", "divider", {}),
      node("aperture.head", "row", { justify: "space-between", align: "end", gap: 20 }, [
        node("aperture.headCopy", "stack", { gap: 6 }, [
          node("aperture.eyebrow", "text", { text: "Week 46 · live", variant: "eyebrow", tone: "accent" }),
          node("aperture.title", "text", { text: "The signal is holding.", variant: "display" }),
          node("aperture.sub", "text", {
            text: "Retention improved for the third week. One segment still needs a decision.",
            variant: "body",
            tone: "muted",
          }),
        ]),
        slotNode("aperture.slot", {
          hint: "Primary action in the workspace header",
          minH: 40,
          role: "action",
          accepts: ["action", "button", "input", "search", "metric", "stat", "number", "card"],
        }),
      ]),
      node("aperture.stats", "grid", { cols: 4, gap: 12 }, [
        node("aperture.stat1", "stat", { label: "Active accounts", value: "12,480", detail: "+8.4% vs last week", tone: "green" }),
        slotNode("aperture.stat2", {
          hint: "KPI card in the stats grid",
          minH: 96,
          role: "metric",
          accepts: ["metric", "stat", "number", "card"],
          // Authored fallback: the studio shows a finished KPI card, and
          // in-context replaces this exact region with the live component.
          children: [
            node("aperture.stat2.fallback", "stat", {
              label: "Expansion",
              value: "31.2%",
              detail: "+2.1 pts",
              tone: "green",
            }),
          ],
        }),
        node("aperture.stat3", "stat", { label: "At risk", value: "184", detail: "12 need review", tone: "amber" }),
        node("aperture.stat4", "stat", { label: "Time to value", value: "4.2d", detail: "−0.6d", tone: "green" }),
      ]),
      node("aperture.body", "grid", { cols: 2, gap: 14 }, [
        node("aperture.chartCard", "card", { pad: 18, elevated: true }, [
          node("aperture.chartHead", "row", { justify: "space-between", align: "center" }, [
            node("aperture.chartTitle", "text", { text: "Retention by cohort", variant: "subtitle" }),
            node("aperture.chartChip", "chip", { text: "12 weeks", tone: "accent" }),
          ]),
          node("aperture.bars", "row", { gap: 7, align: "end", h: 132, pad: "12px 0 0" }, [
            node("aperture.b1", "swatch", { h: 44, tone: "surface", radius: 5, grow: true }),
            node("aperture.b2", "swatch", { h: 58, tone: "surface", radius: 5, grow: true }),
            node("aperture.b3", "swatch", { h: 72, tone: "surface", radius: 5, grow: true }),
            node("aperture.b4", "swatch", { h: 66, tone: "secondary", radius: 5, grow: true }),
            node("aperture.b5", "swatch", { h: 88, tone: "secondary", radius: 5, grow: true }),
            node("aperture.b6", "swatch", { h: 104, tone: "secondary", radius: 5, grow: true }),
            node("aperture.b7", "swatch", { h: 96, tone: "accent", radius: 5, grow: true }),
            node("aperture.b8", "swatch", { h: 122, tone: "accent", radius: 5, grow: true }),
          ]),
          node("aperture.chartFoot", "row", { justify: "space-between" }, [
            node("aperture.chartNote", "text", { text: "Cohort 42 leads at 88% week-4 retention.", variant: "caption", tone: "muted" }),
            node("aperture.chartDelta", "text", { text: "+6.2 pts", variant: "mono", tone: "accent" }),
          ]),
        ]),
        node("aperture.attention", "card", { pad: 18, elevated: true }, [
          node("aperture.attentionHead", "row", { justify: "space-between", align: "center" }, [
            node("aperture.attentionTitle", "text", { text: "Needs a decision", variant: "subtitle" }),
            node("aperture.attentionCount", "chip", { text: "3", tone: "amber" }),
          ]),
          node("aperture.a1", "row", { justify: "space-between", align: "start", pad: "12px 0", border: true }, [
            node("aperture.a1c", "stack", { gap: 3 }, [
              node("aperture.a1t", "text", { text: "Enterprise onboarding stalls at step 3", variant: "label" }),
              node("aperture.a1d", "text", { text: "18 accounts · 9 days", variant: "caption", tone: "muted" }),
            ]),
            node("aperture.a1chip", "chip", { text: "High", tone: "red" }),
          ]),
          node("aperture.a2", "row", { justify: "space-between", align: "start", pad: "12px 0", border: true }, [
            node("aperture.a2c", "stack", { gap: 3 }, [
              node("aperture.a2t", "text", { text: "Mobile invite link expires silently", variant: "label" }),
              node("aperture.a2d", "text", { text: "42 sessions · 3 days", variant: "caption", tone: "muted" }),
            ]),
            node("aperture.a2chip", "chip", { text: "Med", tone: "amber" }),
          ]),
          node("aperture.a3", "row", { justify: "space-between", align: "start", pad: "12px 0" }, [
            node("aperture.a3c", "stack", { gap: 3 }, [
              node("aperture.a3t", "text", { text: "Weekly digest has no empty state", variant: "label" }),
              node("aperture.a3d", "text", { text: "New accounts · 2 days", variant: "caption", tone: "muted" }),
            ]),
            node("aperture.a3chip", "chip", { text: "Low", tone: "ice" }),
          ]),
        ]),
      ]),
      node("aperture.tableCard", "card", { pad: 0, elevated: true }, [
        node("aperture.tableHead", "row", { justify: "space-between", align: "center", pad: "14px 18px" }, [
          node("aperture.tableTitle", "text", { text: "Segments", variant: "subtitle" }),
          node("aperture.tableMeta", "text", { text: "Measured 14m ago", variant: "caption", tone: "muted" }),
        ]),
        node("aperture.rows", "stack", { gap: 0 }, [
          node("aperture.r1", "row", { justify: "space-between", align: "center", pad: "11px 18px", border: true }, [
            node("aperture.r1n", "text", { text: "Product-led · self serve", variant: "body" }),
            node("aperture.r1v", "text", { text: "62%", variant: "mono" }),
            node("aperture.r1m", "meter", { value: 62, tone: "green" }),
          ]),
          node("aperture.r2", "row", { justify: "space-between", align: "center", pad: "11px 18px", border: true }, [
            node("aperture.r2n", "text", { text: "Sales-led · enterprise", variant: "body" }),
            node("aperture.r2v", "text", { text: "27%", variant: "mono" }),
            node("aperture.r2m", "meter", { value: 27, tone: "secondary" }),
          ]),
          node("aperture.r3", "row", { justify: "space-between", align: "center", pad: "11px 18px" }, [
            node("aperture.r3n", "text", { text: "Partner · referral", variant: "body" }),
            node("aperture.r3v", "text", { text: "11%", variant: "mono" }),
            node("aperture.r3m", "meter", { value: 11, tone: "amber" }),
          ]),
        ]),
      ]),
    ]),
  },
  {
    id: "marlow",
    name: "Marlow",
    tagline: "Editorial commerce",
    category: "commerce",
    idea: "A shop that reads like a magazine: one object, one sentence, one invitation.",
    slotHint: "Add-to-bag action on the featured product",
    display: "serif",
    radius: 4,
    palette: {
      bg: "#f6f1e7",
      surface: "#fffdf8",
      surfaceAlt: "#efe7d8",
      ink: "#1d1a16",
      muted: "#7a7268",
      line: "#e3dacb",
      accent: "#7a2e2e",
      accentInk: "#fff8f0",
      accentSoft: "rgba(122,46,46,.08)",
      secondary: "#b98a4b",
    },
    spec: node("marlow.page", "page", { pad: 34 }, [
      node("marlow.nav", "row", { justify: "space-between", align: "center" }, [
        node("marlow.brand", "text", { text: "MARLOW", variant: "mono", tone: "ink" }),
        node("marlow.links", "row", { gap: 20 }, [
          node("marlow.l1", "text", { text: "Objects", variant: "label" }),
          node("marlow.l2", "text", { text: "Journal", variant: "label", tone: "muted" }),
          node("marlow.l3", "text", { text: "Studio", variant: "label", tone: "muted" }),
        ]),
        node("marlow.bag", "chip", { text: "Bag · 2", tone: "accent" }),
      ]),
      node("marlow.rule", "divider", { space: 22 }),
      node("marlow.hero", "grid", { cols: 2, gap: 34, align: "center" }, [
        node("marlow.heroCopy", "stack", { gap: 14 }, [
          node("marlow.eyebrow", "text", { text: "Collection 04 · Autumn", variant: "eyebrow", tone: "secondary" }),
          node("marlow.title", "text", { text: "Objects for a slower home.", variant: "display" }),
          node("marlow.body", "text", {
            text: "Stoneware, oak and linen made in small batches by people we know. Nothing here needs replacing next season.",
            variant: "body",
            tone: "muted",
          }),
          node("marlow.cta", "row", { gap: 10, align: "center" }, [
            node("marlow.ctaPrimary", "button", { text: "Shop the collection", variant: "primary", size: "md" }),
            node("marlow.ctaSecondary", "button", { text: "Read the journal", variant: "ghost", size: "md" }),
          ]),
          node("marlow.proof", "row", { gap: 16, align: "center" }, [
            node("marlow.proof1", "text", { text: "Free returns · 60 days", variant: "caption", tone: "muted" }),
            node("marlow.proof2", "text", { text: "Made in Portugal", variant: "caption", tone: "muted" }),
          ]),
        ]),
        node("marlow.heroArt", "surface", { bg: "surfaceAlt", radius: 4, pad: 0 }, [
          node("marlow.artGlow", "swatch", { h: 300, tone: "secondary", radius: 4, shape: "vase" }),
          node("marlow.artLabel", "text", { text: "The Ash Vase · No. 12", variant: "caption", tone: "ink" }),
        ]),
      ]),
      node("marlow.featured", "card", { pad: 18, elevated: false }, [
        node("marlow.featuredRow", "row", { justify: "space-between", align: "center", gap: 18 }, [
          node("marlow.featuredCopy", "stack", { gap: 4 }, [
            node("marlow.featuredName", "text", { text: "The everyday carafe", variant: "title" }),
            node("marlow.featuredDesc", "text", { text: "Hand-thrown stoneware · 1.2L · Oat glaze", variant: "body", tone: "muted" }),
          ]),
          node("marlow.price", "text", { text: "$86", variant: "price" }),
          slotNode("marlow.slot", {
            hint: "Add-to-bag action on the featured product",
            minH: 40,
            role: "action",
            accepts: ["button", "action", "form", "input"],
          }),
        ]),
      ]),
      node("marlow.grid", "grid", { cols: 3, gap: 16 }, [
        node("marlow.p1", "card", { pad: 12 }, [
          node("marlow.p1art", "swatch", { h: 168, tone: "surfaceAlt", radius: 3 }),
          node("marlow.p1name", "text", { text: "Ash dinner plate", variant: "label" }),
          node("marlow.p1price", "text", { text: "$34", variant: "mono", tone: "muted" }),
        ]),
        node("marlow.p2", "card", { pad: 12 }, [
          node("marlow.p2art", "swatch", { h: 168, tone: "accent", radius: 3 }),
          node("marlow.p2name", "text", { text: "Linen table runner", variant: "label" }),
          node("marlow.p2price", "text", { text: "$58", variant: "mono", tone: "muted" }),
        ]),
        node("marlow.p3", "card", { pad: 12 }, [
          node("marlow.p3art", "swatch", { h: 168, tone: "secondary", radius: 3 }),
          node("marlow.p3name", "text", { text: "Oak serving board", variant: "label" }),
          node("marlow.p3price", "text", { text: "$72", variant: "mono", tone: "muted" }),
        ]),
      ]),
      node("marlow.footer", "row", { justify: "space-between", align: "center", pad: "20px 0 0" }, [
        node("marlow.footNote", "text", { text: "A small studio in Porto. Since 2014.", variant: "caption", tone: "muted" }),
        node("marlow.footLink", "text", { text: "Visit the atelier →", variant: "label", tone: "accent" }),
      ]),
    ]),
  },
  {
    id: "solstice",
    name: "Solstice",
    tagline: "AI workspace",
    category: "ai",
    idea: "A quiet intelligence: one prompt, a short answer, no dashboard noise.",
    slotHint: "Send action in the composer",
    display: "serif",
    radius: 18,
    palette: {
      bg: "#080a16",
      surface: "#101326",
      surfaceAlt: "#161a33",
      ink: "#eef0ff",
      muted: "#9aa0c4",
      line: "#242a4c",
      accent: "#a78bfa",
      accentInk: "#120a2a",
      accentSoft: "rgba(167,139,250,.14)",
      secondary: "#67e8f9",
    },
    spec: node("solstice.page", "page", { pad: 30 }, [
      node("solstice.nav", "row", { justify: "space-between", align: "center" }, [
        node("solstice.brand", "row", { gap: 9, align: "center" }, [
          node("solstice.mark", "swatch", { h: 24, w: 24, tone: "accent", radius: 8, shape: "orb" }),
          node("solstice.brandText", "text", { text: "Solstice", variant: "subtitle" }),
        ]),
        node("solstice.navMid", "row", { gap: 16 }, [
          node("solstice.n1", "text", { text: "Threads", variant: "label", tone: "muted" }),
          node("solstice.n2", "text", { text: "Library", variant: "label", tone: "muted" }),
          node("solstice.n3", "text", { text: "Settings", variant: "label", tone: "muted" }),
        ]),
        node("solstice.avatar", "avatar", { name: "Noor Haddad", size: "sm" }),
      ]),
      node("solstice.hero", "stack", { gap: 10, pad: "54px 0 26px", align: "center" }, [
        node("solstice.eyebrow", "text", { text: "New thread", variant: "eyebrow", tone: "secondary" }),
        node("solstice.title", "text", { text: "What should we think through?", variant: "display", align: "center" }),
        node("solstice.sub", "text", {
          text: "Solstice reads your workspace, then answers with the evidence it used.",
          variant: "body",
          tone: "muted",
          align: "center",
        }),
      ]),
      node("solstice.composer", "card", { pad: 14, elevated: true }, [
        node("solstice.input", "input", { placeholder: "Ask about a customer, a decision, or a page…" }),
        node("solstice.composerRow", "row", { justify: "space-between", align: "center", pad: "12px 0 0" }, [
          node("solstice.chips", "row", { gap: 6 }, [
            node("solstice.c1", "chip", { text: "Summarize churn", tone: "accent" }),
            node("solstice.c2", "chip", { text: "Draft a brief", tone: "muted" }),
            node("solstice.c3", "chip", { text: "Find evidence", tone: "muted" }),
          ]),
          slotNode("solstice.slot", {
            hint: "Send action in the composer",
            minH: 40,
            role: "action",
            accepts: ["button", "action", "form", "input", "text"],
          }),
        ]),
      ]),
      node("solstice.answer", "card", { pad: 20, elevated: true }, [
        node("solstice.answerHead", "row", { justify: "space-between", align: "center" }, [
          node("solstice.answerTitle", "text", { text: "Why enterprise onboarding stalls", variant: "title" }),
          node("solstice.answerChip", "chip", { text: "4 sources", tone: "accent" }),
        ]),
        node("solstice.answerBody", "text", {
          text: "Three of the last five enterprise accounts stopped at the workspace invite step. The invite expires after 24 hours, but the copy promises a week. The fix is a contract change, not a redesign.",
          variant: "body",
          tone: "muted",
        }),
        node("solstice.sources", "stack", { gap: 8, pad: "14px 0 0" }, [
          node("solstice.s1", "row", { gap: 9, align: "center", pad: "9px 11px", border: true }, [
            node("solstice.s1dot", "swatch", { h: 6, w: 6, tone: "secondary", radius: 999 }),
            node("solstice.s1t", "text", { text: "support · 18 conversations", variant: "caption", tone: "muted" }),
            node("solstice.s1v", "text", { text: "open", variant: "mono", tone: "accent" }),
          ]),
          node("solstice.s2", "row", { gap: 9, align: "center", pad: "9px 11px", border: true }, [
            node("solstice.s2dot", "swatch", { h: 6, w: 6, tone: "accent", radius: 999 }),
            node("solstice.s2t", "text", { text: "product · invite contract v3", variant: "caption", tone: "muted" }),
            node("solstice.s2v", "text", { text: "mismatch", variant: "mono", tone: "accent" }),
          ]),
        ]),
      ]),
      node("solstice.suggest", "grid", { cols: 3, gap: 12 }, [
        node("solstice.sg1", "card", { pad: 14 }, [
          node("solstice.sg1t", "text", { text: "Compare cohorts", variant: "label" }),
          node("solstice.sg1d", "text", { text: "Find the segment that changed.", variant: "caption", tone: "muted" }),
        ]),
        node("solstice.sg2", "card", { pad: 14 }, [
          node("solstice.sg2t", "text", { text: "Draft a decision", variant: "label" }),
          node("solstice.sg2d", "text", { text: "Turn this thread into a brief.", variant: "caption", tone: "muted" }),
        ]),
        node("solstice.sg3", "card", { pad: 14 }, [
          node("solstice.sg3t", "text", { text: "Audit a page", variant: "label" }),
          node("solstice.sg3d", "text", { text: "Check a page against its contract.", variant: "caption", tone: "muted" }),
        ]),
      ]),
    ]),
  },
  {
    id: "verdant",
    name: "Verdant",
    tagline: "Climate impact",
    category: "climate",
    idea: "Proof before poetry: the number is the hero, the method is visible.",
    slotHint: "Support action on the project card",
    display: "serif",
    radius: 12,
    palette: {
      bg: "#f2f4ec",
      surface: "#ffffff",
      surfaceAlt: "#e7ecdf",
      ink: "#14251a",
      muted: "#667a6b",
      line: "#dbe3d5",
      accent: "#1f6b45",
      accentInk: "#f4fff8",
      accentSoft: "rgba(31,107,69,.09)",
      secondary: "#c9a227",
    },
    spec: node("verdant.page", "page", { pad: 32 }, [
      node("verdant.nav", "row", { justify: "space-between", align: "center" }, [
        node("verdant.brand", "text", { text: "Verdant", variant: "subtitle" }),
        node("verdant.links", "row", { gap: 18 }, [
          node("verdant.l1", "text", { text: "Projects", variant: "label" }),
          node("verdant.l2", "text", { text: "Method", variant: "label", tone: "muted" }),
          node("verdant.l3", "text", { text: "Ledger", variant: "label", tone: "muted" }),
        ]),
        node("verdant.cta", "button", { text: "Open account", variant: "ghost", size: "sm" }),
      ]),
      node("verdant.rule", "divider", { space: 22 }),
      node("verdant.hero", "grid", { cols: 2, gap: 30, align: "end" }, [
        node("verdant.heroCopy", "stack", { gap: 12 }, [
          node("verdant.eyebrow", "text", { text: "Verified removals · 2026", variant: "eyebrow", tone: "accent" }),
          node("verdant.title", "text", { text: "Twelve thousand tonnes, with receipts.", variant: "display" }),
          node("verdant.body", "text", {
            text: "Every project is measured against a published baseline, audited twice, and retired when the evidence stops holding.",
            variant: "body",
            tone: "muted",
          }),
        ]),
        node("verdant.bigStat", "surface", { bg: "accent", radius: 12, pad: 22 }, [
          node("verdant.bigLabel", "text", { text: "Net removal", variant: "eyebrow", tone: "ink" }),
          node("verdant.bigValue", "text", { text: "12,480 t", variant: "display", tone: "ink" }),
          node("verdant.bigDetail", "text", { text: "▲ 18% vs last year · audited 3d ago", variant: "caption", tone: "ink" }),
        ]),
      ]),
      node("verdant.stats", "grid", { cols: 3, gap: 12 }, [
        node("verdant.st1", "stat", { label: "Active projects", value: "24", detail: "6 new this quarter" }),
        node("verdant.st2", "stat", { label: "Durability", value: "94%", detail: "100-year floor", tone: "green" }),
        node("verdant.st3", "stat", { label: "Cost per tonne", value: "$42", detail: "−$6 vs 2025", tone: "green" }),
      ]),
      node("verdant.projects", "grid", { cols: 2, gap: 14 }, [
        node("verdant.p1", "card", { pad: 16 }, [
          node("verdant.p1art", "swatch", { h: 120, tone: "accent", radius: 8, shape: "wave" }),
          node("verdant.p1name", "text", { text: "Mangrove restoration", variant: "title" }),
          node("verdant.p1desc", "text", { text: "Sundarbans · 2,400 t verified", variant: "caption", tone: "muted" }),
          node("verdant.p1meter", "meter", { label: "Funded", value: 78, tone: "green" }),
          slotNode("verdant.p1slot", {
            hint: "Support action on the project card",
            minH: 40,
            role: "metric",
            accepts: ["metric", "stat", "number", "chart", "button", "action"],
          }),
        ]),
        node("verdant.p2", "card", { pad: 16 }, [
          node("verdant.p2art", "swatch", { h: 120, tone: "secondary", radius: 8, shape: "grid" }),
          node("verdant.p2name", "text", { text: "Biochar at scale", variant: "title" }),
          node("verdant.p2desc", "text", { text: "Andalusia · 1,860 t verified", variant: "caption", tone: "muted" }),
          node("verdant.p2meter", "meter", { label: "Funded", value: 46, tone: "amber" }),
          node("verdant.p2note", "text", { text: "Independent audit due in 12 days", variant: "caption", tone: "muted" }),
        ]),
      ]),
      node("verdant.method", "row", { justify: "space-between", align: "center", pad: "18px 0 0" }, [
        node("verdant.methodText", "text", { text: "Method: baseline → measurement → two audits → retirement.", variant: "caption", tone: "muted" }),
        node("verdant.methodLink", "text", { text: "Read the ledger →", variant: "label", tone: "accent" }),
      ]),
    ]),
  },
  {
    id: "atelier",
    name: "Atelier",
    tagline: "Design portfolio",
    category: "portfolio",
    idea: "The work speaks first; the index is a quiet second voice.",
    slotHint: "Case-study action in the project index",
    display: "serif",
    radius: 2,
    palette: {
      bg: "#f4f1ea",
      surface: "#fbf9f4",
      surfaceAlt: "#e9e3d7",
      ink: "#12100e",
      muted: "#6f6a62",
      line: "#ddd6c9",
      accent: "#12100e",
      accentInk: "#f4f1ea",
      accentSoft: "rgba(18,16,14,.06)",
      secondary: "#b06a3b",
    },
    spec: node("atelier.page", "page", { pad: 38 }, [
      node("atelier.nav", "row", { justify: "space-between", align: "center" }, [
        node("atelier.brand", "text", { text: "ATELIER / 09", variant: "mono" }),
        node("atelier.links", "row", { gap: 20 }, [
          node("atelier.l1", "text", { text: "Work", variant: "label" }),
          node("atelier.l2", "text", { text: "Studio", variant: "label", tone: "muted" }),
          node("atelier.l3", "text", { text: "Contact", variant: "label", tone: "muted" }),
        ]),
      ]),
      node("atelier.hero", "stack", { gap: 16, pad: "46px 0 28px" }, [
        node("atelier.title", "text", { text: "We make digital things feel physical.", variant: "display" }),
        node("atelier.sub", "text", {
          text: "Brand, product and spatial design for people who care how a surface feels.",
          variant: "body",
          tone: "muted",
        }),
      ]),
      node("atelier.feature", "grid", { cols: 2, gap: 16 }, [
        node("atelier.f1", "surface", { bg: "surfaceAlt", radius: 2, pad: 0 }, [
          node("atelier.f1art", "swatch", { h: 300, tone: "secondary", radius: 2, shape: "arch" }),
          node("atelier.f1label", "text", { text: "Kestrel — identity & product", variant: "caption", tone: "ink" }),
        ]),
        node("atelier.f2copy", "stack", { gap: 12, justify: "end" }, [
          node("atelier.f2eyebrow", "text", { text: "Selected work · 2026", variant: "eyebrow", tone: "secondary" }),
          node("atelier.f2title", "text", { text: "A field guide to quiet software.", variant: "title" }),
          node("atelier.f2body", "text", {
            text: "Nine products, one design language. Built with a shared component library so the team could move without losing the thread.",
            variant: "body",
            tone: "muted",
          }),
          slotNode("atelier.f2slot", {
            hint: "Case-study action in the project index",
            minH: 40,
            role: "list",
            accepts: ["list", "table", "card", "button", "action"],
          }),
        ]),
      ]),
      node("atelier.index", "stack", { gap: 0, pad: "22px 0 0" }, [
        node("atelier.i1", "row", { justify: "space-between", align: "center", pad: "14px 0", border: true }, [
          node("atelier.i1n", "text", { text: "Kestrel", variant: "title" }),
          node("atelier.i1m", "text", { text: "Identity · Product · 2026", variant: "caption", tone: "muted" }),
          node("atelier.i1a", "text", { text: "↗", variant: "mono", tone: "accent" }),
        ]),
        node("atelier.i2", "row", { justify: "space-between", align: "center", pad: "14px 0", border: true }, [
          node("atelier.i2n", "text", { text: "Marlow", variant: "title" }),
          node("atelier.i2m", "text", { text: "Commerce · Editorial · 2025", variant: "caption", tone: "muted" }),
          node("atelier.i2a", "text", { text: "↗", variant: "mono", tone: "accent" }),
        ]),
        node("atelier.i3", "row", { justify: "space-between", align: "center", pad: "14px 0" }, [
          node("atelier.i3n", "text", { text: "Verdant", variant: "title" }),
          node("atelier.i3m", "text", { text: "Climate · Data · 2025", variant: "caption", tone: "muted" }),
          node("atelier.i3a", "text", { text: "↗", variant: "mono", tone: "accent" }),
        ]),
      ]),
    ]),
  },
  {
    id: "pulse",
    name: "Pulse",
    tagline: "Health companion",
    category: "health",
    idea: "Care, not compliance: a soft surface that asks one question at a time.",
    slotHint: "Log-today action in the daily card",
    display: "sans",
    radius: 22,
    palette: {
      bg: "#fdf3f1",
      surface: "#ffffff",
      surfaceAlt: "#f8e6e2",
      ink: "#2a1a22",
      muted: "#8a6f79",
      line: "#f0dcd9",
      accent: "#b23a5b",
      accentInk: "#fff8fa",
      accentSoft: "rgba(178,58,91,.09)",
      secondary: "#6b8f71",
    },
    spec: node("pulse.page", "page", { pad: 30 }, [
      node("pulse.nav", "row", { justify: "space-between", align: "center" }, [
        node("pulse.brand", "row", { gap: 8, align: "center" }, [
          node("pulse.mark", "swatch", { h: 22, w: 22, tone: "accent", radius: 999, shape: "orb" }),
          node("pulse.brandText", "text", { text: "Pulse", variant: "subtitle" }),
        ]),
        node("pulse.avatar", "avatar", { name: "Mira Chen", size: "sm" }),
      ]),
      node("pulse.greet", "stack", { gap: 6, pad: "24px 0 8px" }, [
        node("pulse.eyebrow", "text", { text: "Tuesday · 18 November", variant: "eyebrow", tone: "accent" }),
        node("pulse.title", "text", { text: "Good evening, Mira.", variant: "display" }),
        node("pulse.sub", "text", { text: "You have one check-in left today. That is enough.", variant: "body", tone: "muted" }),
      ]),
      node("pulse.grid", "grid", { cols: 2, gap: 14 }, [
        node("pulse.today", "card", { pad: 18 }, [
          node("pulse.todayHead", "row", { justify: "space-between", align: "center" }, [
            node("pulse.todayTitle", "text", { text: "Today's check-in", variant: "title" }),
            node("pulse.todayChip", "chip", { text: "2 min", tone: "accent" }),
          ]),
          node("pulse.todayBody", "text", {
            text: "How is your energy right now? There is no right answer; we are looking for the shape of the week.",
            variant: "body",
            tone: "muted",
          }),
          node("pulse.scale", "row", { gap: 7, pad: "12px 0 0" }, [
            node("pulse.s1", "swatch", { h: 34, grow: true, tone: "surfaceAlt", radius: 10 }),
            node("pulse.s2", "swatch", { h: 34, grow: true, tone: "surfaceAlt", radius: 10 }),
            node("pulse.s3", "swatch", { h: 34, grow: true, tone: "accent", radius: 10 }),
            node("pulse.s4", "swatch", { h: 34, grow: true, tone: "surfaceAlt", radius: 10 }),
            node("pulse.s5", "swatch", { h: 34, grow: true, tone: "surfaceAlt", radius: 10 }),
          ]),
          slotNode("pulse.slot", {
            hint: "Log-today action in the daily card",
            minH: 40,
            role: "list",
            accepts: ["list", "table", "metric", "stat", "button", "action"],
          }),
        ]),
        node("pulse.week", "card", { pad: 18 }, [
          node("pulse.weekHead", "row", { justify: "space-between", align: "center" }, [
            node("pulse.weekTitle", "text", { text: "This week", variant: "title" }),
            node("pulse.weekChip", "chip", { text: "Steady", tone: "green" }),
          ]),
          node("pulse.weekStat", "stat", { label: "Energy", value: "7.4", detail: "▲ 0.6 vs last week", tone: "green" }),
          node("pulse.weekMeter", "meter", { label: "Sleep", value: 82, tone: "green" }),
          node("pulse.weekMeter2", "meter", { label: "Movement", value: 64, tone: "amber" }),
          node("pulse.weekNote", "text", { text: "Two short walks beat one long one for you.", variant: "caption", tone: "muted" }),
        ]),
      ]),
      node("pulse.habits", "card", { pad: 18 }, [
        node("pulse.habitsHead", "row", { justify: "space-between", align: "center" }, [
          node("pulse.habitsTitle", "text", { text: "Small things that helped", variant: "title" }),
          node("pulse.habitsMeta", "text", { text: "Last 14 days", variant: "caption", tone: "muted" }),
        ]),
        node("pulse.h1", "row", { justify: "space-between", align: "center", pad: "11px 0", border: true }, [
          node("pulse.h1t", "text", { text: "Morning light before screens", variant: "body" }),
          node("pulse.h1v", "text", { text: "11 / 14", variant: "mono", tone: "accent" }),
        ]),
        node("pulse.h2", "row", { justify: "space-between", align: "center", pad: "11px 0", border: true }, [
          node("pulse.h2t", "text", { text: "Walk after lunch", variant: "body" }),
          node("pulse.h2v", "text", { text: "9 / 14", variant: "mono", tone: "accent" }),
        ]),
        node("pulse.h3", "row", { justify: "space-between", align: "center", pad: "11px 0" }, [
          node("pulse.h3t", "text", { text: "No screens after 22:00", variant: "body" }),
          node("pulse.h3v", "text", { text: "4 / 14", variant: "mono", tone: "muted" }),
        ]),
      ]),
    ]),
  },
  {
    id: "harbor",
    name: "Harbor",
    tagline: "Onboarding",
    category: "onboarding",
    idea: "A threshold, not a funnel: one promise on the left, one step on the right.",
    slotHint: "Continue action in the onboarding step",
    display: "serif",
    radius: 18,
    palette: {
      bg: "#0f172a",
      surface: "#16213b",
      surfaceAlt: "#1d2a48",
      ink: "#f1f5fb",
      muted: "#93a1bd",
      line: "#27334f",
      accent: "#4f8cff",
      accentInk: "#ffffff",
      accentSoft: "rgba(79,140,255,.14)",
      secondary: "#5eead4",
    },
    spec: node("harbor.page", "page", { pad: 30 }, [
      node("harbor.brand", "row", { gap: 9, align: "center" }, [
        node("harbor.mark", "swatch", { h: 24, w: 24, tone: "accent", radius: 8, shape: "grid" }),
        node("harbor.brandText", "text", { text: "Harbor", variant: "subtitle" }),
      ]),
      node("harbor.split", "grid", { cols: 2, gap: 30, align: "center", pad: "28px 0 0" }, [
        node("harbor.copy", "stack", { gap: 14 }, [
          node("harbor.eyebrow", "text", { text: "Step 2 of 4", variant: "eyebrow", tone: "secondary" }),
          node("harbor.title", "text", { text: "Bring your team into the same room.", variant: "display" }),
          node("harbor.body", "text", {
            text: "Harbor keeps decisions, evidence and the people who own them in one place. Invite the first three people; you can change this later.",
            variant: "body",
            tone: "muted",
          }),
          node("harbor.steps", "stack", { gap: 10, pad: "8px 0 0" }, [
            node("harbor.st1", "row", { gap: 10, align: "center" }, [
              node("harbor.st1dot", "swatch", { h: 10, w: 10, tone: "accent", radius: 999 }),
              node("harbor.st1t", "text", { text: "Workspace created", variant: "label", tone: "muted" }),
            ]),
            node("harbor.st2", "row", { gap: 10, align: "center" }, [
              node("harbor.st2dot", "swatch", { h: 10, w: 10, tone: "secondary", radius: 999 }),
              node("harbor.st2t", "text", { text: "Invite your team", variant: "label", tone: "ink" }),
            ]),
            node("harbor.st3", "row", { gap: 10, align: "center" }, [
              node("harbor.st3dot", "swatch", { h: 10, w: 10, tone: "line", radius: 999 }),
              node("harbor.st3t", "text", { text: "Connect your first source", variant: "label", tone: "muted" }),
            ]),
          ]),
        ]),
        node("harbor.card", "card", { pad: 22, elevated: true }, [
          node("harbor.cardTitle", "text", { text: "Invite your team", variant: "title" }),
          node("harbor.cardBody", "text", { text: "They will get a link that expires in 7 days.", variant: "caption", tone: "muted" }),
          node("harbor.input1", "input", { placeholder: "name@company.com" }),
          node("harbor.input2", "input", { placeholder: "name@company.com" }),
          node("harbor.roleRow", "row", { justify: "space-between", align: "center" }, [
            node("harbor.roleLabel", "text", { text: "Role", variant: "caption", tone: "muted" }),
            node("harbor.roleChip", "chip", { text: "Editor", tone: "accent" }),
          ]),
          slotNode("harbor.slot", {
            hint: "Continue action in the onboarding step",
            minH: 40,
            role: "action",
            accepts: ["button", "action", "form", "input"],
          }),
          node("harbor.skip", "text", { text: "Skip for now", variant: "caption", tone: "muted", align: "center" }),
        ]),
      ]),
      node("harbor.quote", "card", { pad: 18 }, [
        node("harbor.quoteText", "text", {
          text: "“We stopped losing decisions in chat. The evidence is attached to the thing it belongs to.”",
          variant: "body",
          tone: "ink",
        }),
        node("harbor.quoteBy", "row", { gap: 9, align: "center" }, [
          node("harbor.quoteAvatar", "avatar", { name: "Dana Ruiz", size: "xs" }),
          node("harbor.quoteName", "text", { text: "Dana Ruiz · Northwind", variant: "caption", tone: "muted" }),
        ]),
      ]),
    ]),
  },
  {
    id: "nocturne-ops",
    name: "Nocturne Ops",
    tagline: "Command center",
    category: "operations",
    idea: "Dense, legible, unromantic: the page tells you what is on fire and what to do.",
    slotHint: "Run-check action in the incident header",
    display: "sans",
    radius: 8,
    palette: {
      bg: "#0a090c",
      surface: "#141118",
      surfaceAlt: "#1b1720",
      ink: "#f5efe6",
      muted: "#9c9088",
      line: "#2b2531",
      accent: "#ffb35c",
      accentInk: "#1a1108",
      accentSoft: "rgba(255,179,92,.12)",
      secondary: "#8fd8ff",
    },
    spec: node("ops.page", "page", { pad: 24 }, [
      node("ops.nav", "row", { justify: "space-between", align: "center" }, [
        node("ops.brand", "row", { gap: 8, align: "center" }, [
          node("ops.mark", "swatch", { h: 20, w: 20, tone: "accent", radius: 6, shape: "orb" }),
          node("ops.brandText", "text", { text: "NOCTURNE / OPS", variant: "mono" }),
        ]),
        node("ops.status", "row", { gap: 8, align: "center" }, [
          node("ops.statusChip", "chip", { text: "2 degraded", tone: "amber" }),
          node("ops.statusTime", "text", { text: "02:14 UTC", variant: "mono", tone: "muted" }),
        ]),
      ]),
      node("ops.rule", "divider", { space: 16 }),
      node("ops.head", "row", { justify: "space-between", align: "end", gap: 18 }, [
        node("ops.headCopy", "stack", { gap: 5 }, [
          node("ops.eyebrow", "text", { text: "Incident 241 · open", variant: "eyebrow", tone: "accent" }),
          node("ops.title", "text", { text: "Checkout latency is climbing.", variant: "title" }),
          node("ops.sub", "text", { text: "p95 1.8s · 4 regions · started 18m ago", variant: "mono", tone: "muted" }),
        ]),
        slotNode("ops.slot", {
          hint: "Run-check action in the incident header",
          minH: 40,
          role: "action",
          accepts: ["button", "action", "status", "metric", "stat"],
        }),
      ]),
      node("ops.kpis", "grid", { cols: 4, gap: 10 }, [
        node("ops.k1", "stat", { label: "Error rate", value: "0.42%", detail: "▲ 0.11", tone: "amber" }),
        node("ops.k2", "stat", { label: "p95", value: "1.8s", detail: "SLO 1.2s", tone: "red" }),
        node("ops.k3", "stat", { label: "Throughput", value: "1.2k/s", detail: "stable", tone: "green" }),
        node("ops.k4", "stat", { label: "Queue", value: "184", detail: "draining", tone: "green" }),
      ]),
      node("ops.grid", "grid", { cols: 2, gap: 12 }, [
        node("ops.services", "card", { pad: 0 }, [
          node("ops.servicesHead", "row", { justify: "space-between", align: "center", pad: "12px 14px" }, [
            node("ops.servicesTitle", "text", { text: "Services", variant: "label" }),
            node("ops.servicesMeta", "text", { text: "12 checked", variant: "caption", tone: "muted" }),
          ]),
          node("ops.svc1", "row", { justify: "space-between", align: "center", pad: "10px 14px", border: true }, [
            node("ops.svc1n", "text", { text: "checkout-api", variant: "mono" }),
            node("ops.svc1s", "chip", { text: "degraded", tone: "amber" }),
          ]),
          node("ops.svc2", "row", { justify: "space-between", align: "center", pad: "10px 14px", border: true }, [
            node("ops.svc2n", "text", { text: "payments-gateway", variant: "mono" }),
            node("ops.svc2s", "chip", { text: "healthy", tone: "green" }),
          ]),
          node("ops.svc3", "row", { justify: "space-between", align: "center", pad: "10px 14px", border: true }, [
            node("ops.svc3n", "text", { text: "inventory", variant: "mono" }),
            node("ops.svc3s", "chip", { text: "healthy", tone: "green" }),
          ]),
          node("ops.svc4", "row", { justify: "space-between", align: "center", pad: "10px 14px" }, [
            node("ops.svc4n", "text", { text: "search-index", variant: "mono" }),
            node("ops.svc4s", "chip", { text: "degraded", tone: "amber" }),
          ]),
        ]),
        node("ops.timeline", "card", { pad: 14 }, [
          node("ops.timelineHead", "row", { justify: "space-between", align: "center" }, [
            node("ops.timelineTitle", "text", { text: "Timeline", variant: "label" }),
            node("ops.timelineMeta", "text", { text: "auto", variant: "caption", tone: "muted" }),
          ]),
          node("ops.e1", "row", { gap: 9, align: "start", pad: "10px 0", border: true }, [
            node("ops.e1dot", "swatch", { h: 6, w: 6, tone: "accent", radius: 999 }),
            node("ops.e1t", "text", { text: "02:14 · p95 crossed 1.5s in eu-west", variant: "caption", tone: "muted" }),
          ]),
          node("ops.e2", "row", { gap: 9, align: "start", pad: "10px 0", border: true }, [
            node("ops.e2dot", "swatch", { h: 6, w: 6, tone: "amber", radius: 999 }),
            node("ops.e2t", "text", { text: "02:08 · retry budget exhausted", variant: "caption", tone: "muted" }),
          ]),
          node("ops.e3", "row", { gap: 9, align: "start", pad: "10px 0" }, [
            node("ops.e3dot", "swatch", { h: 6, w: 6, tone: "line", radius: 999 }),
            node("ops.e3t", "text", { text: "01:56 · deploy 8f2c1a rolled out", variant: "caption", tone: "muted" }),
          ]),
        ]),
      ]),
      node("ops.runbook", "card", { pad: 14 }, [
        node("ops.runbookHead", "row", { justify: "space-between", align: "center" }, [
          node("ops.runbookTitle", "text", { text: "Next action", variant: "label" }),
          node("ops.runbookChip", "chip", { text: "runbook · checkout-latency", tone: "ice" }),
        ]),
        node("ops.runbookBody", "text", {
          text: "Roll back 8f2c1a in eu-west, then drain the retry queue. Expected recovery 4 minutes. Evidence is attached to the incident.",
          variant: "caption",
          tone: "muted",
        }),
        node("ops.runbookActions", "row", { gap: 8, align: "center" }, [
          node("ops.runbookPrimary", "button", { text: "Open runbook", variant: "primary", size: "sm" }),
          node("ops.runbookSecondary", "button", { text: "Acknowledge", variant: "ghost", size: "sm" }),
        ]),
      ]),
    ]),
  },
];

export function templateById(id: string | null | undefined): Template | undefined {
  if (!id) return undefined;
  return TEMPLATES.find((template) => template.id === id);
}

/**
 * The shelf is authored, not user-supplied, so an empty shelf is a build
 * defect rather than a runtime state. This keeps the fallback type honest.
 */
export const DEFAULT_TEMPLATE: Template =
  TEMPLATES[0] ??
  (() => {
    throw new Error("The template shelf is empty; at least one template is required.");
  })();

export const TEMPLATE_CATEGORIES: Array<{ id: Template["category"] | "all"; label: string }> = [
  { id: "all", label: "All" },
  { id: "analytics", label: "Analytics" },
  { id: "commerce", label: "Commerce" },
  { id: "ai", label: "AI" },
  { id: "climate", label: "Climate" },
  { id: "portfolio", label: "Portfolio" },
  { id: "health", label: "Health" },
  { id: "onboarding", label: "Onboarding" },
  { id: "operations", label: "Operations" },
];
