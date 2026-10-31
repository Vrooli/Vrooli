import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState } from "react";
import { recipientsClient } from "../../api/notifications";

const COPY: Record<string, { title: string; emptyDraftTitle: string; emptyDraftLead: string; emptyDraftBody: string; restoreSavedOrder: string; saveEmptyOrder: string; savedOrderStill: string; eyebrow: string; body: string; unsupportedTitle: string; unsupportedEyebrow: string; add: string; save: string; review: string; saved: string; unknown: string; unknownLead: string; unknownRetained: string; keepDraft: string; policy: string; unknownChannel: string; reviewRequired: string; reviewUnsupported: string; clear: string; clearPrompt: string; clearTitle: string; currentOrder: string; clearOutcome: string; clearStaged: string; clearStagedBody: string; clearImpact: string; confirmClear: string; keepEditing: string; cancel: string; remove: string; moveUp: string; moveDown: string; empty: string; loading: string; readError: string; retry: string; saving: string; savedOrder: string; unsavedOrder: string; stepTiming: string; noChannels: string; open: string; back: string; editOrder: string; previouslyLoaded: string; attemptedOrder: string; checkSaved: string; unknownTitle: string; confirmedTitle: string; notAppliedTitle: string; uncertainTitle: string }> = {
  en: { title: "Escalation order", emptyDraftTitle: "No escalation channels", emptyDraftLead: "The cleared order is not saved yet", emptyDraftBody: "Overdue critical asks will have no additional channel in this chain after you save.", restoreSavedOrder: "Restore saved order", saveEmptyOrder: "Save empty order", savedOrderStill: "Your saved order is still {order}.", eyebrow: "Channel order · unsaved draft", body: "Try these channels in sequence for an eligible request.", unsupportedTitle: "Review this saved channel", unsupportedEyebrow: "Escalation order · unsupported entry", add: "Add channel", save: "Save order", review: "Review this order before saving", saved: "Order saved and confirmed from the latest read.", unknown: "Read the saved order before making another change.", unknownLead: "The save result is unknown", unknownRetained: "Your attempted order is retained. No automatic retry or change to quiet hours has been made.", keepDraft: "Keep draft", policy: "Only overdue critical asks use this order. Non-critical asks expire unless a separately governed reversible default applies.", unknownChannel: "Unrecognized saved identifier. Preserved unchanged; target availability is unknown.", reviewRequired: "A target needs review", reviewUnsupported: "Saving stays unavailable until unsupported or missing targets are resolved. No saved entry was silently removed.", clear: "Clear order…", clearPrompt: "This will stage an empty order. Eligibility rules do not change. Save only after reviewing the result.", clearTitle: "Clear escalation order?", currentOrder: "Current order", clearOutcome: "Overdue critical asks will have no additional channel in this chain.", clearStaged: "Clear stages an empty order", clearStagedBody: "Save order will persist the change. Quiet hours stay separate.", clearImpact: "This does not retract notifications or establish an exact delivery or expiry time.", confirmClear: "Clear this draft order", keepEditing: "Keep editing", cancel: "Cancel", remove: "Remove", moveUp: "Move up", moveDown: "Move down", empty: "No channels are currently ordered.", loading: "Loading escalation order…", readError: "Escalation order could not be read.", retry: "Try again", saving: "Saving…", savedOrder: "Saved order", unsavedOrder: "Unsaved draft", stepTiming: "Timing is fixed at 5 minutes per response step; target availability and content approval are checked separately.", noChannels: "No channels", open: "Edit order", back: "Back to Settings", editOrder: "Back to order", previouslyLoaded: "Previously loaded", attemptedOrder: "Attempted order", checkSaved: "Check saved order", unknownTitle: "Order not confirmed", confirmedTitle: "Order saved", notAppliedTitle: "Order was not applied", uncertainTitle: "Saved order changed" },
  ja: { title: "エスカレーション順", emptyDraftTitle: "エスカレーション先はありません", emptyDraftLead: "クリアした順序はまだ保存されていません", emptyDraftBody: "保存後、期限を過ぎた重大な依頼にはこの連鎖の追加通知先がなくなります。", restoreSavedOrder: "保存済みの順序に戻す", saveEmptyOrder: "空の順序を保存", savedOrderStill: "保存済みの順序は引き続き{order}です。", eyebrow: "エスカレーション順 · 未保存の下書き", body: "通知先を順番に試します。", unsupportedTitle: "保存済みの通知先を確認", unsupportedEyebrow: "エスカレーション順 · 未対応の項目", add: "通知先を追加", save: "順序を保存", review: "保存前に順序を確認", saved: "保存を確認しました。最新の順序を読み込みました。", unknown: "次の変更を行う前に保存済みの順序を確認してください。", unknownLead: "保存結果を確認できません。", unknownRetained: "試した順序は下書きとして保持されています。自動再試行も静かな時間の変更も行っていません。", keepDraft: "下書きを保持", policy: "期限超過の重大な依頼のみがこの順序を使用します。非重大な依頼は、別途管理される取消可能な既定対応が適用されない限り期限切れになります。", unknownChannel: "認識できない保存済み識別子です。変更せずに保持しました。利用可否は不明です。", reviewRequired: "通知先の確認が必要です", reviewUnsupported: "未対応または見つからない通知先を解決するまで保存できません。保存済みの通知先は自動では削除されません。", clear: "順序をクリア…", clearPrompt: "空の順序を保存用に準備します。対象条件は変更しません。結果を確認してから保存してください。", clearTitle: "エスカレーション順をクリアしますか？", currentOrder: "現在の順序", clearOutcome: "期限を過ぎた重大な依頼には、この連鎖の追加通知先がなくなります。", clearStaged: "クリアすると空の順序が準備されます", clearStagedBody: "変更は「順序を保存」で反映されます。静かな時間の設定は別です。", clearImpact: "通知の取り消しや、正確な配信・期限の設定は行われません。", confirmClear: "この下書きの順序をクリア", keepEditing: "編集を続ける", cancel: "キャンセル", remove: "削除", moveUp: "上へ移動", moveDown: "下へ移動", empty: "順序に設定された通知先はありません。", loading: "順序を読み込み中…", readError: "順序を読み込めませんでした。", retry: "再試行", saving: "保存中…", savedOrder: "保存済みの順序", unsavedOrder: "未保存の下書き", stepTiming: "応答の間隔は5分固定です。通知先の利用可否と内容の承認は別途確認されます。", noChannels: "通知先なし", open: "順序を編集", back: "設定に戻る", editOrder: "順序に戻る", previouslyLoaded: "読み込み済み", attemptedOrder: "保存しようとした順序", checkSaved: "保存済みの順序を確認", unknownTitle: "順序を確認できません", confirmedTitle: "順序を保存しました", notAppliedTitle: "順序は反映されませんでした", uncertainTitle: "保存済みの順序が異なります" },
  ar: { title: "ترتيب التصعيد", emptyDraftTitle: "لا توجد قنوات تصعيد", emptyDraftLead: "لم يتم حفظ الترتيب الممسوح بعد", emptyDraftBody: "لن يكون للطلبات الحرجة المتأخرة قناة إضافية في هذه السلسلة بعد الحفظ.", restoreSavedOrder: "استعادة الترتيب المحفوظ", saveEmptyOrder: "حفظ الترتيب الفارغ", savedOrderStill: "لا يزال الترتيب المحفوظ هو {order}.", eyebrow: "ترتيب القنوات · مسودة غير محفوظة", body: "جرّب هذه القنوات بالتتابع لطلب مؤهل.", unsupportedTitle: "راجع هذه القناة المحفوظة", unsupportedEyebrow: "ترتيب التصعيد · عنصر غير مدعوم", add: "إضافة قناة", save: "حفظ الترتيب", review: "راجع الترتيب قبل الحفظ", saved: "تم تأكيد الترتيب من أحدث قراءة.", unknown: "اقرأ الترتيب المحفوظ قبل إجراء تغيير آخر.", unknownLead: "نتيجة الحفظ غير معروفة.", unknownRetained: "تم الاحتفاظ بالترتيب الذي حاولت حفظه كمسودة. لم تحدث إعادة محاولة تلقائية أو أي تغيير في ساعات الهدوء.", keepDraft: "الاحتفاظ بالمسودة", policy: "ينطبق هذا الترتيب على الطلبات الحرجة المتأخرة فقط. تنتهي الطلبات غير الحرجة ما لم ينطبق خيار افتراضي قابل للعكس يخضع لإدارة منفصلة.", unknownChannel: "المعرّف المحفوظ غير معروف. تم الاحتفاظ به دون تغيير؛ ومدى توفر الهدف غير معروف.", reviewRequired: "تحتاج إحدى الوجهات إلى المراجعة", reviewUnsupported: "سيظل الحفظ غير متاح حتى تُحل الوجهات غير المدعومة أو المفقودة. لم تتم إزالة أي وجهة محفوظة بصمت.", clear: "مسح الترتيب…", clearPrompt: "سيتم تجهيز ترتيب فارغ. لن تتغير قواعد الأهلية. راجع النتيجة قبل الحفظ.", clearTitle: "هل تريد مسح ترتيب التصعيد؟", currentOrder: "الترتيب الحالي", clearOutcome: "لن يكون للطلبات الحرجة المتأخرة قناة إضافية في هذه السلسلة.", clearStaged: "سيجهّز المسح ترتيبًا فارغًا", clearStagedBody: "سيحفظ تغيير الترتيب عند اختيار حفظ الترتيب. تظل ساعات الهدوء منفصلة.", clearImpact: "لا يؤدي ذلك إلى سحب الإشعارات أو تحديد وقت دقيق للتسليم أو الانتهاء.", confirmClear: "مسح مسودة الترتيب هذه", keepEditing: "متابعة التعديل", cancel: "إلغاء", remove: "إزالة", moveUp: "نقل لأعلى", moveDown: "نقل لأسفل", empty: "لا توجد قنوات مرتبة حاليًا.", loading: "جارٍ تحميل الترتيب…", readError: "تعذّرت قراءة الترتيب.", retry: "حاول مرة أخرى", saving: "جارٍ الحفظ…", savedOrder: "الترتيب المحفوظ", unsavedOrder: "مسودة غير محفوظة", stepTiming: "التوقيت ثابت عند 5 دقائق لكل خطوة؛ ويتم التحقق بشكل منفصل من توفر الهدف وموافقة المحتوى.", noChannels: "لا توجد قنوات", open: "تعديل الترتيب", back: "العودة إلى الإعدادات", editOrder: "العودة إلى الترتيب", previouslyLoaded: "المحمّل سابقًا", attemptedOrder: "الترتيب المطلوب", checkSaved: "التحقق من الترتيب المحفوظ", unknownTitle: "لم يتم تأكيد الترتيب", confirmedTitle: "تم حفظ الترتيب", notAppliedTitle: "لم يُطبّق الترتيب", uncertainTitle: "الترتيب المحفوظ مختلف" },
};
const ADD_CHANNEL_COPY: Record<string, { title: string; eyebrow: string; body: string; configuration: string; availabilityTitle: string; availabilityBody: string; back: string }> = {
  en: { title: "Add a channel", eyebrow: "Escalation order · {channel} selected", body: "Append a supported channel to the order.", configuration: "Target configuration must be checked before saving", availabilityTitle: "Availability is checked separately", availabilityBody: "An adapter-ready observation does not prove a deliverable destination or permit sensitive content.", back: "Back to order" },
  ja: { title: "通知先を追加", eyebrow: "エスカレーション順 · {channel} を選択中", body: "対応している通知先を順序に追加します。", configuration: "保存前に通知先の設定を確認してください", availabilityTitle: "利用可否は別途確認されます", availabilityBody: "アダプターの準備状況だけでは、配信可能な宛先や機密情報の送信許可を証明できません。", back: "順序に戻る" },
  ar: { title: "إضافة قناة", eyebrow: "ترتيب التصعيد · {channel} محدد", body: "أضف قناة مدعومة إلى الترتيب.", configuration: "يجب التحقق من إعداد الوجهة قبل الحفظ", availabilityTitle: "يتم التحقق من التوفر بشكل منفصل", availabilityBody: "لا تثبت ملاحظة جاهزية المهايئ وجود وجهة قابلة للتسليم أو السماح بالمحتوى الحساس.", back: "العودة إلى الترتيب" },
};
const ORDER_HEADER_COPY: Record<string, Record<"editor" | "review" | "add" | "outcome", { title: string; back: string }>> = {
  en: {
    editor: { title: "Channel order", back: "Settings" },
    review: { title: "Escalation order", back: "Order" },
    add: { title: "Escalation order", back: "Order" },
    outcome: { title: "Escalation", back: "Order" },
  },
  ja: {
    editor: { title: "チャンネル順", back: "設定" },
    review: { title: "エスカレーション順", back: "順序" },
    add: { title: "エスカレーション順", back: "順序" },
    outcome: { title: "エスカレーション", back: "順序" },
  },
  ar: {
    editor: { title: "ترتيب القنوات", back: "الإعدادات" },
    review: { title: "ترتيب التصعيد", back: "الترتيب" },
    add: { title: "ترتيب التصعيد", back: "الترتيب" },
    outcome: { title: "التصعيد", back: "الترتيب" },
  },
};
const labels: Record<string, string> = { email: "Email", webhook: "Webhook", web_push: "Web Push", browser: "Browser", push: "Push" };
type OrderOutcome = { kind: "confirmed" | "notApplied" | "uncertain" | "unknown"; saved: string[]; attempted: string[] };

export function EscalationOrderPanel({ focused = false, onFocusedChange, onEditorHeaderChange }: { focused?: boolean; onFocusedChange?: (focused: boolean) => void; onEditorHeaderChange?: (header: { title: string; backLabel: string; onBack: () => void } | null) => void }) {
  const lang = (document.documentElement.lang || "en").split("-")[0] || "en"; const c = COPY[lang] ?? COPY.en!;
  const headerCopy = ORDER_HEADER_COPY[lang] ?? ORDER_HEADER_COPY.en!;
  const queryClient = useQueryClient();
  const devices = useQuery({ queryKey: ["recipient-devices"], queryFn: () => recipientsClient.listDevices({}), refetchOnWindowFocus: true });
  const chain = useQuery({ queryKey: ["escalation-chain"], queryFn: () => recipientsClient.getEscalationChain({}), refetchOnWindowFocus: true });
  const [order, setOrder] = useState<string[] | undefined>(undefined); const [addingChannel, setAddingChannel] = useState(false); const [outcome, setOutcome] = useState<OrderOutcome | null>(null); const [saving, setSaving] = useState(false); const [checking, setChecking] = useState(false); const [confirmClear, setConfirmClear] = useState(false);
  const clearReviewHeading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { if (confirmClear) clearReviewHeading.current?.focus(); }, [confirmClear]);
  useEffect(() => { if (chain.data && order === undefined) setOrder(chain.data.steps.map((step) => step.channel)); }, [chain.data, order]);
  const shownOrder = order ?? chain.data?.steps.map((step) => step.channel) ?? [];
  const savedOrder = chain.data?.steps.map((step) => step.channel) ?? [];
  const available = useMemo(() => [...new Set((devices.data?.devices ?? []).flatMap((device) => device.channels))], [devices.data]);
  const unsupportedChannels = devices.isSuccess ? shownOrder.filter((channel) => !available.includes(channel)) : [];
  const reviewingUnsupported = unsupportedChannels.length > 0;
  const headerState = outcome ? "outcome" : addingChannel ? "add" : reviewingUnsupported ? "review" : "editor";
  const activeHeader = headerCopy[headerState];
  useEffect(() => {
    if (!onEditorHeaderChange) return undefined;
    if (!focused) {
      onEditorHeaderChange(null);
      return () => onEditorHeaderChange(null);
    }
    onEditorHeaderChange({
      title: activeHeader.title,
      backLabel: activeHeader.back,
      onBack: () => {
        let selector: string;
        if (headerState === "outcome") {
          setOutcome(null);
          selector = '[data-testid="escalation-order-edit"]';
        } else if (headerState === "add") {
          setAddingChannel(false);
          selector = '[data-testid="escalation-order-add-select"]';
        } else {
          onFocusedChange?.(false);
          selector = '[data-testid="escalation-order-open"]';
        }
        const restoreFocus = () => document.querySelector<HTMLElement>(selector)?.focus({ preventScroll: true });
        if (typeof window.requestAnimationFrame === "function") window.requestAnimationFrame(restoreFocus);
        else window.setTimeout(restoreFocus, 0);
      },
    });
    return () => onEditorHeaderChange(null);
  }, [focused, activeHeader.back, activeHeader.title, headerState, onEditorHeaderChange, onFocusedChange]);
  const catalogReady = devices.isSuccess && unsupportedChannels.length === 0;
  const addChannelCopy = ADD_CHANNEL_COPY[lang] ?? ADD_CHANNEL_COPY.en!;
  const dirty = JSON.stringify(shownOrder) !== JSON.stringify((chain.data?.steps ?? []).map((step) => step.channel));
  const emptyDraft = dirty && shownOrder.length === 0;
  const savedOrderLabel = savedOrder.map((channel) => labels[channel] ?? channel).join(", then ");
  const changeOrder = (change: (current: string[]) => string[]) => { onFocusedChange?.(true); setOutcome(null); setOrder((current) => change(current ?? shownOrder)); };
  const move = (index: number, delta: number) => changeOrder((current) => { const next = [...current]; const target = index + delta; if (target < 0 || target >= next.length) return next; [next[index], next[target]] = [next[target]!, next[index]!]; return next; });
  const cancelDraft = () => { setOrder([...savedOrder]); setConfirmClear(false); setOutcome(null); onFocusedChange?.(false); };
  const save = async () => {
    onFocusedChange?.(true); setSaving(true); const attempted = [...shownOrder]; const before = [...savedOrder];
    try { await recipientsClient.setEscalationChain({ channels: attempted }); } catch { /* The readback below is authoritative. */ }
    try {
      await queryClient.cancelQueries({ queryKey: ["escalation-chain"], exact: true });
      const latest = await queryClient.fetchQuery({ queryKey: ["escalation-chain"], queryFn: () => recipientsClient.getEscalationChain({}), staleTime: 0 });
      const actual = latest.steps.map((step) => step.channel);
      const kind: OrderOutcome["kind"] = JSON.stringify(actual) === JSON.stringify(attempted) ? "confirmed" : JSON.stringify(actual) === JSON.stringify(before) ? "notApplied" : "uncertain";
      setOrder(kind === "confirmed" ? actual : attempted);
      setOutcome({ kind, saved: actual, attempted });
    } catch { setOutcome({ kind: "unknown", saved: before, attempted }); }
    setSaving(false);
  };
  const checkSaved = async () => {
    if (!outcome || checking) return;
    setChecking(true);
    try {
      await queryClient.cancelQueries({ queryKey: ["escalation-chain"], exact: true });
      const latest = await queryClient.fetchQuery({ queryKey: ["escalation-chain"], queryFn: () => recipientsClient.getEscalationChain({}), staleTime: 0 });
      const actual = latest.steps.map((step) => step.channel);
      const kind: OrderOutcome["kind"] = JSON.stringify(actual) === JSON.stringify(outcome.attempted) ? "confirmed" : JSON.stringify(actual) === JSON.stringify(outcome.saved) ? "notApplied" : "uncertain";
      setOrder(kind === "confirmed" ? actual : outcome.attempted);
      setOutcome({ ...outcome, kind, saved: actual });
    } catch { setOutcome({ ...outcome, kind: "unknown" }); }
    setChecking(false);
  };
  if (focused && !outcome && confirmClear) return <section data-testid="escalation-order-root" className="flex min-h-[min(72vh,44rem)] flex-col gap-4 rounded-none border-0 p-0" aria-labelledby="escalation-clear-heading">
    <div><p className="text-sm text-app-muted-foreground">{c.title}</p><h2 ref={clearReviewHeading} tabIndex={-1} id="escalation-clear-heading" className="mt-3 text-2xl font-bold">{c.clearTitle}</h2></div>
    <dl className="flex flex-col gap-3"><div className="flex justify-between gap-4"><dt className="text-app-muted-foreground">{c.currentOrder}</dt><dd className="text-end">{shownOrder.length ? shownOrder.map((channel) => labels[channel] ?? channel).join(" → ") : c.noChannels}</dd></div></dl>
    <p>{c.clearOutcome}</p>
    <aside className="border-s-4 border-app-warning ps-4 py-3"><h3 className="font-semibold">{c.clearStaged}</h3><p className="mt-2">{c.clearStagedBody}</p><p data-testid="escalation-order-policy" className="mt-2 text-sm text-app-muted-foreground">{c.policy}</p></aside>
    <p>{c.clearImpact}</p>
    <div className="sm:sticky sm:bottom-0 mt-auto grid grid-cols-2 gap-3 border-t border-app-border bg-app-background py-3"><button data-testid="escalation-order-cancel-clear" type="button" onClick={() => setConfirmClear(false)} className="min-h-12 rounded-lg bg-app-primary px-4 font-semibold text-white">{c.keepEditing}</button><button data-testid="escalation-order-confirm-clear" type="button" onClick={() => { setOutcome(null); setOrder([]); setConfirmClear(false); }} className="min-h-12 rounded-lg border border-app-border px-4">{c.confirmClear}</button></div>
  </section>;
  if (focused && !outcome && addingChannel) return <section data-testid="escalation-order-root" className="flex flex-col gap-3 rounded-none border-0 p-0" aria-labelledby="escalation-add-channel-heading">
    <div>
      <p className="text-sm text-app-muted-foreground">{addChannelCopy.eyebrow.replace("{channel}", labels[shownOrder.at(-1) ?? ""] ?? shownOrder.at(-1) ?? c.noChannels)}</p>
      <h2 id="escalation-add-channel-heading" className="mt-2 text-3xl font-bold">{addChannelCopy.title}</h2>
      <p className="text-lg">{addChannelCopy.body}</p>
    </div>
    <div data-testid="escalation-order-channel-picker" className="flex flex-col">
      {available.filter((channel) => !shownOrder.includes(channel)).map((channel) => <button key={channel} data-testid={`escalation-order-channel-choice-${channel}`} type="button" onClick={() => { changeOrder((current) => [...current, channel]); setAddingChannel(false); }} className="relative flex items-center border-b border-app-border py-4 text-start">
        <span><span className="block text-xl font-semibold">{labels[channel] ?? channel}</span><span className="mt-1 block text-sm text-app-muted-foreground">{addChannelCopy.configuration}</span></span>
        <span aria-hidden="true" className="absolute end-0 top-1/2 -translate-y-1/2 text-xl text-app-muted-foreground">›</span>
      </button>)}
      {!available.filter((channel) => !shownOrder.includes(channel)).length && <p className="py-5 text-sm text-app-muted-foreground">{c.noChannels}</p>}
    </div>
    <aside className="border-s-4 ps-4 py-3" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}>
      <h3 className="font-semibold">{addChannelCopy.availabilityTitle}</h3>
      <p className="mt-2 text-[0.8125rem]">{addChannelCopy.availabilityBody}</p>
    </aside>
    <div className="mt-4 border-t border-app-border bg-app-background py-3">
      <button data-testid="escalation-order-add-back" type="button" onClick={() => setAddingChannel(false)} className="min-h-11 w-full rounded-lg border border-app-border px-4">{addChannelCopy.back}</button>
    </div>
  </section>;
  return <section data-testid="escalation-order-root" className={`flex ${focused ? "min-h-0 flex-1 flex-col gap-0" : "flex-col gap-4"} rounded-none border-0 p-0`} aria-labelledby="escalation-order-heading">
    {!focused && <>
      <div><h3 id="escalation-order-heading" className="text-lg font-semibold">{c.title}</h3><p className="text-sm text-app-muted-foreground">{c.body}</p><p data-testid="escalation-order-policy" className="mt-2 border-s-4 border-app-warning ps-3 text-sm">{c.policy}</p><button data-testid="escalation-order-open" type="button" onClick={() => onFocusedChange?.(true)} className="mt-3 min-h-11 rounded-lg bg-app-primary px-4 font-medium text-white">{c.open}</button></div>
      {chain.isLoading ? <p role="status">{c.loading}</p> : <div className="rounded-lg bg-app-surface-raised p-3 text-sm"><p><span className="font-medium">{c.savedOrder}:</span> {savedOrder.length ? savedOrder.map((channel) => labels[channel] ?? channel).join(" → ") : c.noChannels}</p></div>}
    </>}
    {focused && !outcome && <div className="flex min-h-0 flex-1 flex-col">
    <div data-testid="escalation-order-scroll-region" className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto overscroll-contain pb-2 pe-2">
    <div className={reviewingUnsupported ? "pb-4" : ""}><div className="flex items-center justify-between gap-3"><p className="text-sm text-app-muted-foreground">{reviewingUnsupported ? c.unsupportedEyebrow : c.eyebrow}</p>{!reviewingUnsupported && !onEditorHeaderChange && <button type="button" onClick={() => onFocusedChange?.(false)} className="min-h-11 rounded px-2 text-sm text-app-muted-foreground underline underline-offset-4">{c.back}</button>}</div><h2 id="escalation-order-heading" className={`mt-2 font-bold ${reviewingUnsupported ? "whitespace-nowrap text-[1.625rem] leading-tight" : "text-3xl"}`}>{reviewingUnsupported ? c.unsupportedTitle : emptyDraft ? c.emptyDraftTitle : c.title}</h2>{!reviewingUnsupported && !emptyDraft && <p className="mt-3 text-lg">{c.body}</p>}<p data-testid="escalation-order-policy" className="mt-2 border-s-4 border-app-warning ps-3 text-sm">{c.policy}</p></div>
    {chain.isLoading && <p role="status">{c.loading}</p>}
    {chain.isError && <div role="alert"><p>{c.readError}</p><button type="button" onClick={() => void chain.refetch()}>{c.retry}</button></div>}
    {emptyDraft ? <aside data-testid="escalation-order-empty-draft" className="border-s-4 border-app-warning ps-4 py-3"><h3 className="font-semibold">{c.emptyDraftLead}</h3><p className="mt-2">{c.emptyDraftBody}</p></aside> : <ol data-testid="escalation-order-list" className="flex flex-col gap-2" aria-label={c.title}>
      {shownOrder.map((channel, index) => <li key={`${channel}-${index}`} data-testid={`escalation-order-channel-${channel}`} className={`flex flex-col gap-3 border-b border-app-border ${reviewingUnsupported ? "py-4" : "py-5"}`}>
        <div><h3 className="text-xl font-semibold">{index + 1}. {labels[channel] ?? channel}</h3><p className="mt-2 text-sm text-app-muted-foreground">5 minute response step</p></div>
        {!available.includes(channel) && <span className="text-sm text-app-warning">{c.unknownChannel}</span>}
        <div className="grid min-w-0 grid-cols-3 gap-2">
          <button data-testid={`escalation-order-up-${channel}`} type="button" aria-label={`${c.moveUp} ${channel}`} disabled={index === 0} onClick={() => move(index, -1)} className="min-h-11 min-w-0 rounded border border-app-border px-1 text-xs leading-tight sm:px-2 sm:text-sm disabled:opacity-50">{c.moveUp}</button>
          <button data-testid={`escalation-order-down-${channel}`} type="button" aria-label={`${c.moveDown} ${channel}`} disabled={index === shownOrder.length - 1} onClick={() => move(index, 1)} className="min-h-11 min-w-0 rounded border border-app-border px-1 text-xs leading-tight sm:px-2 sm:text-sm disabled:opacity-50">{c.moveDown}</button>
          <button data-testid={`escalation-order-remove-${channel}`} type="button" aria-label={`${c.remove} ${channel}`} onClick={() => changeOrder((current) => current.filter((_, i) => i !== index))} className="min-h-11 min-w-0 rounded border border-app-border px-1 text-xs leading-tight sm:px-2 sm:text-sm">{c.remove}</button>
        </div>
      </li>)}
      {!shownOrder.length && <li className="rounded border border-dashed border-app-border p-4 text-sm text-app-muted-foreground">{c.empty}</li>}
    </ol>}
    {unsupportedChannels.length > 0 && <aside data-testid="escalation-order-review-required" role="alert" className="border-s-4 border-app-warning ps-4 py-3"><h3 className="font-semibold">{c.reviewRequired}</h3><p className="mt-2">{c.reviewUnsupported}</p></aside>}
    {!reviewingUnsupported && !addingChannel && <button data-testid="escalation-order-add-select" type="button" onClick={() => setAddingChannel(true)} className="min-h-11 w-full rounded border border-app-border px-4">{c.add}</button>}
    {emptyDraft && savedOrder.length > 0 && <button data-testid="escalation-order-restore-saved" type="button" onClick={() => changeOrder(() => [...savedOrder])} className="min-h-11 w-full rounded border border-app-border px-4">{c.restoreSavedOrder}</button>}
    {!reviewingUnsupported && shownOrder.length > 0 && <button data-testid="escalation-order-clear" type="button" onClick={() => setConfirmClear(true)} className="min-h-11 w-full rounded border border-app-border px-4">{c.clear}</button>}
    {!reviewingUnsupported && <div className="text-sm leading-6">
      <p>{emptyDraft ? c.savedOrderStill.replace("{order}", savedOrderLabel) : <><span className="font-medium">{c.savedOrder}:</span> {savedOrder.length ? savedOrderLabel : c.noChannels}</>}</p>
      {!emptyDraft && <p className="mt-1 text-app-muted-foreground">{c.stepTiming}</p>}
    </div>}
    </div>
    {(dirty || reviewingUnsupported) && <div className="shrink-0 grid grid-cols-2 gap-3 border-t border-app-border bg-app-background py-3">
      <button data-testid="escalation-order-save" type="button" disabled={saving || chain.isLoading || chain.isError || !catalogReady || addingChannel} onClick={() => void save()} className="min-h-12 rounded-lg bg-app-primary px-4 py-2 font-semibold text-white disabled:opacity-50">{saving ? c.saving : emptyDraft ? c.saveEmptyOrder : c.save}</button>
      <button data-testid="escalation-order-cancel" type="button" disabled={saving} onClick={emptyDraft ? () => onFocusedChange?.(false) : cancelDraft} className="min-h-12 rounded-lg border border-app-border px-4">{reviewingUnsupported ? c.keepDraft : emptyDraft ? c.keepEditing : c.cancel}</button>
    </div>}
    </div>}
    {focused && outcome && <>
      <div><p className="text-sm text-app-muted-foreground">Escalation · {outcome.kind === "unknown" ? "response not received" : "latest read"}</p><h2 id="escalation-order-heading" className="mt-3 text-2xl font-bold">{outcome.kind === "confirmed" ? c.confirmedTitle : outcome.kind === "notApplied" ? c.notAppliedTitle : outcome.kind === "unknown" ? c.unknownTitle : c.uncertainTitle}</h2></div>
      <aside data-testid="escalation-order-message" role="status" aria-live="polite" className="border-s-4 border-app-warning ps-4 py-3">
        <h3 className="font-semibold">{outcome.kind === "confirmed" ? c.confirmedTitle : outcome.kind === "notApplied" ? c.notAppliedTitle : outcome.kind === "unknown" ? c.unknownLead : c.uncertainTitle}</h3>
        <p className="mt-2">{outcome.kind === "confirmed" ? c.saved : outcome.kind === "unknown" ? c.unknown : outcome.kind === "notApplied" ? "The latest read still shows the previously saved order. No automatic retry was made." : "The latest read differs from both orders. Review it before deciding what to do."}</p>
      </aside>
      <dl className="flex flex-col gap-4"><div className="flex justify-between gap-4"><dt className="text-app-muted-foreground">{c.previouslyLoaded}</dt><dd className="text-end">{outcome.saved.length ? outcome.saved.map((channel) => labels[channel] ?? channel).join(" → ") : c.noChannels}</dd></div><div className="flex justify-between gap-4"><dt className="text-app-muted-foreground">{c.attemptedOrder}</dt><dd className="text-end">{outcome.attempted.length ? outcome.attempted.map((channel) => labels[channel] ?? channel).join(" → ") : c.noChannels}</dd></div></dl>
      {outcome.kind === "unknown" && <p data-testid="escalation-order-unknown-retained">{c.unknownRetained}</p>}
      <div className="sm:sticky sm:bottom-0 flex flex-wrap gap-3 border-t border-app-border bg-app-background py-3"><button data-testid="escalation-order-check-saved" type="button" onClick={() => void checkSaved()} disabled={checking || outcome.kind === "confirmed"} className="min-h-11 flex-1 rounded-lg bg-app-primary px-4 text-white disabled:opacity-50">{checking ? c.loading : c.checkSaved}</button><button data-testid="escalation-order-edit" type="button" onClick={() => setOutcome(null)} className="min-h-11 flex-1 rounded-lg border border-app-border px-4">{outcome.kind === "unknown" ? c.keepDraft : c.editOrder}</button></div>
    </>}
  </section>;
}
