import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { recipientsClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";

type Draft = { enabled: boolean; start: string; end: string; timezone: string; criticalOverride: boolean };
type Screen = "week" | "day" | "review" | "result";
type ChangedDay = { weekday: number; before: Draft | null; requested: Draft | null; attempted: boolean };
type OutcomeRow = ChangedDay & { state: "confirmed" | "notApplied" | "notAttempted" | "uncertain" | "unknown"; actual: Draft | null };
type SaveOutcome = { kind: "saved" | "partial" | "unknown"; rows: OutcomeRow[]; followUpReadFailed?: boolean };
type TimezoneReviewCopy = {
  eyebrow: string; title: string; newZone: string; keepLocal: string; keepLocalBody: string;
  before: string; draft: string; onlyDay: string; onlyDayBody: string; stage: string; review: string;
};
type Copy = {
  title: string; weekly: string; explanation: string; save: string; savedTitle: string; saved: string; equalTitle: string; equalAlertTitle: string; equalAlertBody: string; equalInvalid: string; unsavedDraft: string; dayTitle: string; timezoneChangeTitle: string; timezoneChangeBody: string; timezoneBefore: string; timezoneDraft: string;
  timeTitle: string; timeAlertTitle: string; timeAlertBody: string; timeFieldHint: string; timezoneTitle: string; timezoneAlertTitle: string; timezoneAlertBody: string; timezoneFieldHint: string;
  overnightTitle: string; overnightAlertTitle: string; overnightAlertBody: string; startsOn: string; endsNextDay: string;
  saveOvernight: string;
  partialTitle: string; partialReviewTitle: string; partialUnknownTitle: string; partialUnknownEyebrow: string; partialUnknownStatusTitle: string; partialUnknownStatus: string; partialUnknownAdvice: string; reviewSavedValues: string; partial: string; unknownTitle: string; unknown: string; overnight: string; invalid: string;
  review: string; reviewTitle: string; reviewBody: string; replaceManyTitle: string; replaceManyBody: string; criticalManyOff: string; criticalManyOn: string; criticalQuietTitle: string; weekendUnchanged: string; remainingDraftRetained: string; critical: string; criticalDisclosure: string; criticalOn: string; criticalOff: string;
  day: string; starts: string; ends: string; timezone: string; enabled: string; loading: string; readError: string;
  retry: string; saving: string; removing: string; differentZones: string; differentZonesBody: string;
  noWindow: string; edit: string; done: string; cancel: string; remove: string; removeTitle: string;
  removePrompt: string; removalStagedTitle: string; removalStagedBody: string; stageRemoval: string; removeAction: string; keepWindow: string; reviewDays: string; back: string; criticalUrgency: string; criticalStaysQuiet: string;
  appliesDay: string; confirmed: string; stillUncertain: string; notApplied: string; notAttempted: string; uncertain: string;
  unknownState: string; requested: string; latestRead: string; checkSaved: string; keepEditing: string;
  returnToWeek: string; keepDraft: string; savedWindow: string; addWindow: string; updatesStopped: string;
  doneStages: string;
};
const COPY: Record<string, Copy> = {
  en: {
    title: "Quiet hours", weekly: "Weekly windows", explanation: "Set one same-day window per weekday. Each day saves separately; check the latest read after a save.", save: "Save quiet hours", savedTitle: "Quiet hours saved", saved: "Every changed day is confirmed from the latest read.", equalTitle: "Choose different times", equalAlertTitle: "Start and end are the same", equalAlertBody: "Choose different times. Equal times do not mean an all-day quiet window.", equalInvalid: "Choose different start and end times.", unsavedDraft: "unsaved draft", dayTitle: "{day}’s quiet window", timezoneChangeTitle: "Time zone changed", timezoneChangeBody: "Local clock hours stay the same; the absolute time can shift.", timezoneBefore: "Before", timezoneDraft: "Draft", timeTitle: "Check Monday’s time", timeAlertTitle: "The start time isn’t valid", timeAlertBody: "Use 24-hour time from 00:00 to 23:59. No changes have been saved.", timeFieldHint: "Enter a time from 00:00 to 23:59.", timezoneTitle: "Check Monday’s time zone", timezoneAlertTitle: "The time zone isn’t recognized", timezoneAlertBody: "Enter an IANA time-zone name. No changes have been saved.", timezoneFieldHint: "Enter a recognized IANA time zone.", overnightTitle: "Overnight window isn’t ready", overnightAlertTitle: "An updated scheduling contract is needed", overnightAlertBody: "This installation cannot safely apply Monday night through Tuesday morning. Keep this draft or choose a same-day window.", startsOn: "starts", endsNextDay: "ends", saveOvernight: "Save overnight window", partialTitle: "Some quiet hours need checking", partialReviewTitle: "The week is only partly confirmed", partialUnknownTitle: "{day} is still unknown", partialUnknownEyebrow: "Quiet hours · saved-window check", partialUnknownStatusTitle: "The saved windows couldn’t be read", partialUnknownStatus: "{day}’s write may have completed. Later days were not sent or retried.", partialUnknownAdvice: "Keep the draft while you check again. Don’t replace the entire week to resolve one unknown result.", reviewSavedValues: "Review day-by-day values", partial: "Read the saved windows before sending another change.", unknownTitle: "Quiet hours save result is unknown", unknown: "The latest read could not be confirmed. Your attempted values are retained; no automatic retry was made.", overnight: "Overnight windows are held and cannot be saved here. Existing values remain visible; no timing conversion is applied.", invalid: "Use valid, different times and a recognized timezone. End time must be later than start time for this same-day editor.", review: "Review changes before saving", reviewTitle: "Review quiet-hour changes", reviewBody: "Only the named weekdays will change. Each window keeps its own time zone; other days stay as saved.", replaceManyTitle: "Replace {count} saved windows?", replaceManyBody: "Each changed day will use {window}.", criticalManyOff: "The override is off for all {count} days.", criticalManyOn: "The critical override is on for all changed days.", criticalQuietTitle: "Critical urgency will also stay quiet.", weekendUnchanged: "Saturday and Sunday are unchanged.", remainingDraftRetained: "Your remaining draft is retained.", critical: "Let critical urgency through", criticalDisclosure: "The critical urgency override is separate from permission to access notifications carrying the Critical content label.", criticalOn: "Critical notifications will also be quiet.", criticalOff: "Critical notifications can still come through.", day: "Day", starts: "Start", ends: "End", timezone: "Time zone", enabled: "Set a quiet window", loading: "Loading quiet hours…", readError: "Quiet hours could not be read.", retry: "Try again", saving: "Saving…", removing: "window will be removed after you save.", differentZones: "Different time zones", differentZonesBody: "Each window uses its own saved zone.", noWindow: "No quiet window", edit: "Edit", done: "Done", cancel: "Cancel", remove: "Remove window", removeTitle: "Remove {day}’s window?", removePrompt: "{day} will no longer have this quiet window after you save the changed week.", removalStagedTitle: "Removal is staged first", removalStagedBody: "Other weekdays are unchanged. Keep editing to retain {day}’s saved window.", stageRemoval: "Stage removal", removeAction: "Remove {day}’s window", keepWindow: "Keep window", reviewDays: "Review changed days", back: "Week", appliesDay: "Applies to {day} only.", confirmed: "Confirmed", stillUncertain: "Still uncertain", notApplied: "Not applied", notAttempted: "Not attempted", uncertain: "Uncertain", unknownState: "Unknown", requested: "Requested", latestRead: "Latest saved value", checkSaved: "Check saved windows", keepEditing: "Keep editing", returnToWeek: "Return to week", keepDraft: "Keep draft", savedWindow: "Saved window", criticalUrgency: "Critical urgency", criticalStaysQuiet: "Stays quiet", addWindow: "Add quiet window", updatesStopped: "Saving stopped at the first uncertain day. Later days were not sent.", doneStages: "Done stages {day}’s edit. Save quiet hours reviews and saves changed days."
  },
  ja: {
    title: "通知を控える時間", weekly: "週間設定", explanation: "曜日ごとに同日内の時間帯を設定します。各曜日は個別に保存され、保存後に最新の内容を確認します。", save: "通知を控える時間を保存", savedTitle: "通知を控える時間を保存しました", saved: "変更した曜日を最新の読み取りで確認しました。", equalTitle: "異なる時刻を選択してください", equalAlertTitle: "開始と終了が同じです", equalAlertBody: "異なる時刻を選んでください。同じ時刻は終日通知を控える設定ではありません。", equalInvalid: "開始時刻と終了時刻を異なる時刻にしてください。", unsavedDraft: "未保存の下書き", dayTitle: "{day}の通知を停止する時間帯", timezoneChangeTitle: "タイムゾーンを変更しました", timezoneChangeBody: "表示時刻は維持されます。基準となる実際の時刻は変わる場合があります。", timezoneBefore: "変更前", timezoneDraft: "下書き", timeTitle: "時刻を確認してください", timeAlertTitle: "有効な時刻を入力してください", timeAlertBody: "00:00 から 23:59 までの24時間表記を使用してください。", timeFieldHint: "24時間表記で有効な時刻を入力してください。", timezoneTitle: "タイムゾーンを確認してください", timezoneAlertTitle: "タイムゾーンを認識できません", timezoneAlertBody: "IANAタイムゾーン名を入力してください。変更は保存されていません。", timezoneFieldHint: "認識できるIANAタイムゾーンを入力してください。", overnightTitle: "翌日にまたがる設定は未対応です", overnightAlertTitle: "スケジュール仕様の更新が必要です", overnightAlertBody: "この設定は翌日に終了します。編集を続けるかキャンセルしてください。時刻の変更は保存されていません。", startsOn: "開始", endsNextDay: "翌日終了", saveOvernight: "翌日またぎを保存", partialTitle: "一部の曜日を確認してください", partialReviewTitle: "週の設定は一部のみ確認できました", partialUnknownTitle: "{day} はまだ不明です", partialUnknownEyebrow: "静穏時間 · 保存済みウィンドウの確認", partialUnknownStatusTitle: "保存済みの時間帯を読み取れませんでした", partialUnknownStatus: "{day} の書き込みは完了した可能性があります。後続の日は送信も再試行もしていません。", partialUnknownAdvice: "確認を続ける間は下書きを保持してください。不明な結果が1件あるだけで、週間設定全体を置き換えないでください。", reviewSavedValues: "曜日ごとの保存値を確認", partial: "保存済みの時間帯を確認してから、次の変更を送信してください。", unknownTitle: "保存結果を確認できません", unknown: "最新の読み取りを確認できませんでした。入力値は保持され、自動再試行は行っていません。", overnight: "翌日にまたがる時間帯は保留中のため保存できません。既存値を表示し、時間の変換は行いません。", invalid: "有効な異なる時刻とタイムゾーンを指定してください。同日設定では終了時刻を開始時刻より後にします。", review: "保存前に変更を確認", reviewTitle: "通知を控える時間の変更を確認", reviewBody: "変更する曜日だけが更新されます。各曜日のタイムゾーンは維持され、ほかの曜日はそのままです。", replaceManyTitle: "保存済みの時間帯を{count}件置き換えますか？", replaceManyBody: "変更する曜日は{window}になります。", criticalManyOff: "変更する{count}曜日で緊急通知の例外はオフです。", criticalManyOn: "変更するすべての曜日で緊急通知の例外はオンです。", criticalQuietTitle: "緊急通知も通知を控える時間の対象です。", weekendUnchanged: "土曜日と日曜日は変更されません。", remainingDraftRetained: "残りの下書きは保持されています。", critical: "緊急度が最も高い通知をこの時間帯も許可する", criticalDisclosure: "緊急度の例外は、通知内容の「Critical」ラベルへのアクセス許可とは別の設定です。", criticalOn: "緊急通知も控えられます。", criticalOff: "緊急通知は引き続き受け取れます。", day: "曜日", starts: "通知を停止し始める時刻", ends: "通知の停止を終了する時刻", timezone: "この曜日に適用するタイムゾーン", enabled: "通知を控える時間を設定", loading: "設定を読み込み中…", readError: "通知を控える時間を読み込めませんでした。", retry: "再試行", saving: "保存中…", removing: "の設定は保存後に削除されます。", differentZones: "異なるタイムゾーン", differentZonesBody: "各時間帯は個別に保存されたタイムゾーンを使います。", noWindow: "設定なし", edit: "編集", done: "完了", cancel: "キャンセル", remove: "時間帯を削除", removeTitle: "{day}の時間帯を削除しますか？", removePrompt: "変更した週を保存すると、{day}のこの通知停止時間はなくなります。", removalStagedTitle: "削除は先に下書きとして反映されます", removalStagedBody: "ほかの曜日は変更されません。編集を続けると{day}の保存済み時間帯を保持できます。", stageRemoval: "削除を下書きにする", removeAction: "{day}の時間帯を削除", keepWindow: "時間帯を残す", reviewDays: "変更した曜日を確認", back: "週間設定", appliesDay: "{day}のみ適用されます。", confirmed: "確認済み", stillUncertain: "まだ不明", notApplied: "未反映", notAttempted: "未実行", uncertain: "確認できません", unknownState: "不明", requested: "リクエスト", latestRead: "最新の保存値", checkSaved: "保存済み設定を確認", keepEditing: "下書きを保持", returnToWeek: "週間設定へ戻る", keepDraft: "下書きを保持", savedWindow: "保存済み時間帯", criticalUrgency: "緊急度", criticalStaysQuiet: "通知を控えます", addWindow: "時間帯を追加", updatesStopped: "最初に結果を確認できなかった曜日で保存を停止しました。後続の曜日は送信していません。", doneStages: "「完了」で{day}の変更を下書きに反映します。保存する前に、変更した曜日を確認できます。"
  },
  ar: {
    title: "ساعات الهدوء", weekly: "النوافذ الأسبوعية", explanation: "حدّد فترة في اليوم نفسه لكل يوم. يُحفظ كل يوم على حدة، ثم نتحقق من أحدث قراءة.", save: "حفظ ساعات الهدوء", savedTitle: "تم حفظ ساعات الهدوء", saved: "تم تأكيد كل يوم تغيّر من أحدث قراءة.", equalTitle: "اختر وقتين مختلفين", equalAlertTitle: "وقت البدء والانتهاء متطابق", equalAlertBody: "اختر وقتين مختلفين. لا تعني الأوقات المتطابقة كتم الإشعارات طوال اليوم.", equalInvalid: "اختر وقتي بدء وانتهاء مختلفين.", unsavedDraft: "مسودة غير محفوظة", dayTitle: "ساعات الهدوء لـ {day}", timezoneChangeTitle: "تم تغيير المنطقة الزمنية", timezoneChangeBody: "تبقى ساعات الساعة المحلية كما هي؛ وقد يتغير الوقت المطلق.", timezoneBefore: "قبل", timezoneDraft: "المسودة", timeTitle: "تحقق من الوقت", timeAlertTitle: "أدخل وقتًا صالحًا", timeAlertBody: "استخدم تنسيق 24 ساعة من 00:00 إلى 23:59.", timeFieldHint: "أدخل وقتًا صالحًا بتنسيق 24 ساعة.", timezoneTitle: "تحقق من المنطقة الزمنية", timezoneAlertTitle: "المنطقة الزمنية غير معروفة", timezoneAlertBody: "أدخل اسم منطقة زمنية وفق IANA. لم يتم حفظ أي تغييرات.", timezoneFieldHint: "أدخل منطقة زمنية معروفة وفق IANA.", overnightTitle: "الفترة الليلية غير مدعومة بعد", overnightAlertTitle: "يلزم تحديث عقد الجدولة", overnightAlertBody: "تبدأ هذه الفترة في يوم وتنتهي في اليوم التالي. تابع التعديل أو ألغِه؛ لم تُحفظ أي تغييرات زمنية.", startsOn: "تبدأ", endsNextDay: "تنتهي في اليوم التالي", saveOvernight: "حفظ الفترة الليلية", partialTitle: "بعض الأيام تحتاج إلى مراجعة", partialReviewTitle: "لم يتم تأكيد إعدادات الأسبوع بالكامل", partialUnknownTitle: "ما زال {day} غير معروف", partialUnknownEyebrow: "ساعات الهدوء · التحقق من النوافذ المحفوظة", partialUnknownStatusTitle: "تعذرت قراءة النوافذ المحفوظة", partialUnknownStatus: "ربما اكتملت كتابة {day}. لم تُرسل الأيام اللاحقة أو يُعاد إرسالها.", partialUnknownAdvice: "احتفظ بالمسودة أثناء إعادة التحقق. لا تستبدل الأسبوع كله لمعالجة نتيجة واحدة غير معروفة.", reviewSavedValues: "مراجعة القيم المحفوظة لكل يوم", partial: "تحقق من الفترات المحفوظة قبل إرسال أي تغيير آخر.", unknownTitle: "نتيجة حفظ ساعات الهدوء غير معروفة", unknown: "تعذّر تأكيد أحدث قراءة. احتفظنا بالقيم المطلوبة ولم نعد المحاولة تلقائيًا.", overnight: "الفترات التي تمتد إلى اليوم التالي معلّقة ولا يمكن حفظها هنا. تبقى القيم الحالية ظاهرة دون تحويل.", invalid: "استخدم وقتين مختلفين صالحين ومنطقة زمنية معروفة. يجب أن يكون وقت الانتهاء لاحقًا في اليوم نفسه.", review: "راجع التغييرات قبل الحفظ", reviewTitle: "راجع تغييرات ساعات الهدوء", reviewBody: "ستتغير الأيام المذكورة فقط. تحتفظ كل فترة بمنطقتها الزمنية، وتبقى الأيام الأخرى كما هي.", replaceManyTitle: "استبدال {count} فترات محفوظة؟", replaceManyBody: "سيستخدم كل يوم متغير {window}.", criticalManyOff: "الاستثناء متوقف لكل الأيام المتغيرة وعددها {count}.", criticalManyOn: "استثناء الإشعارات الحرجة مفعّل لكل الأيام المتغيرة.", criticalQuietTitle: "ستُكتم الإشعارات الحرجة أيضًا.", weekendUnchanged: "لن تتغير إعدادات السبت والأحد.", remainingDraftRetained: "تم الاحتفاظ ببقية المسودة.", critical: "السماح بالإشعارات الحرجة خلال هذه الفترة", criticalDisclosure: "استثناء درجة الإلحاح منفصل عن السماح بالوصول إلى محتوى الإشعار المصنف «Critical».", criticalOn: "سيتم إسكات الإشعارات الحرجة أيضًا.", criticalOff: "ستظل الإشعارات الحرجة تصل إليك.", day: "اليوم", starts: "تبدأ", ends: "تنتهي", timezone: "المنطقة الزمنية", enabled: "تفعيل ساعات الهدوء", loading: "جارٍ تحميل ساعات الهدوء…", readError: "تعذّرت قراءة ساعات الهدوء.", retry: "حاول مرة أخرى", saving: "جارٍ الحفظ…", removing: "ستُزال الفترة بعد الحفظ.", differentZones: "مناطق زمنية مختلفة", differentZonesBody: "تستخدم كل فترة منطقتها الزمنية المحفوظة بشكل مستقل.", noWindow: "لا توجد فترة هدوء", edit: "تعديل", done: "تم", cancel: "إلغاء", remove: "إزالة الفترة", removeTitle: "إزالة فترة {day}؟", removePrompt: "لن تكون فترة الهدوء هذه موجودة ليوم {day} بعد حفظ تغييرات الأسبوع.", removalStagedTitle: "تُجهّز الإزالة أولًا", removalStagedBody: "لن تتغير الأيام الأخرى. واصل التحرير للاحتفاظ بفترة {day} المحفوظة.", stageRemoval: "تجهيز الإزالة", removeAction: "إزالة فترة {day}", keepWindow: "الاحتفاظ بالفترة", reviewDays: "مراجعة الأيام المتغيرة", back: "الأسبوع", appliesDay: "تُطبق على {day} فقط.", confirmed: "تم التأكيد", stillUncertain: "غير مؤكد", notApplied: "لم تُطبق", notAttempted: "لم تُرسل", uncertain: "تعذّر التأكيد", unknownState: "غير معروف", requested: "المطلوب", latestRead: "أحدث قيمة محفوظة", checkSaved: "التحقق من الفترات المحفوظة", keepEditing: "متابعة التعديل", returnToWeek: "العودة إلى الأسبوع", keepDraft: "الاحتفاظ بالمسودة", savedWindow: "الفترة المحفوظة", criticalUrgency: "الإشعارات الحرجة", criticalStaysQuiet: "ستظل مكتومة", addWindow: "إضافة فترة هدوء", updatesStopped: "توقف الحفظ عند أول يوم تعذّر تأكيده. لم تُرسل الأيام اللاحقة.", doneStages: "يحوّل زر «تم» تعديل يوم {day} إلى مسودة. راجع الأيام المتغيرة قبل حفظها."
  },
};
const DAYS: Record<string, string[]> = { en: ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"], ja: ["日曜日", "月曜日", "火曜日", "水曜日", "木曜日", "金曜日", "土曜日"], ar: ["الأحد", "الاثنين", "الثلاثاء", "الأربعاء", "الخميس", "الجمعة", "السبت"] };
const TIMEZONE_REVIEW_COPY: Record<string, TimezoneReviewCopy> = {
  en: { eyebrow: "{day} · recurring quiet window", title: "{day}’s time zone", newZone: "New time zone", keepLocal: "Keep local clock hours", keepLocalBody: "{start}–{end} stays {start}–{end} in the new zone. Its absolute time can change.", before: "Before", draft: "Draft", onlyDay: "Only {day} changes", onlyDayBody: "Other weekdays keep their saved zones. Review this change before saving.", stage: "Stage time-zone change", review: "Review time-zone change" },
  ja: { eyebrow: "{day} · 繰り返し設定", title: "{day}のタイムゾーン", newZone: "新しいタイムゾーン", keepLocal: "現地の時刻を維持", keepLocalBody: "新しい地域でも{start}〜{end}のままです。絶対時刻は変わる場合があります。", before: "変更前", draft: "変更案", onlyDay: "変更するのは{day}のみ", onlyDayBody: "他の曜日の保存済みタイムゾーンは変わりません。保存前にこの変更を確認してください。", stage: "タイムゾーン変更を反映", review: "タイムゾーン変更を確認" },
  ar: { eyebrow: "{day} · فترة هدوء متكررة", title: "المنطقة الزمنية لـ{day}", newZone: "المنطقة الزمنية الجديدة", keepLocal: "الإبقاء على ساعات الوقت المحلي", keepLocalBody: "تبقى {start}–{end} كما هي في المنطقة الجديدة. قد يتغير الوقت المطلق.", before: "قبل", draft: "المسودة", onlyDay: "يتغير {day} فقط", onlyDayBody: "تبقى المناطق الزمنية المحفوظة للأيام الأخرى كما هي. راجع هذا التغيير قبل الحفظ.", stage: "تجهيز تغيير المنطقة الزمنية", review: "مراجعة تغيير المنطقة الزمنية" },
};

function emptyDraft(): Draft { return { enabled: false, start: "22:00", end: "08:00", timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC", criticalOverride: false }; }
function asDraft(row: { start: string; end: string; timezone: string; criticalOverride: boolean }): Draft { return { enabled: true, start: row.start, end: row.end, timezone: row.timezone, criticalOverride: row.criticalOverride }; }
function validTime(value: string) { return /^([01]\d|2[0-3]):[0-5]\d$/.test(value); }
function sameWindow(a: Draft | null, b: Draft | null) {
  if (a === null || b === null) return a === b;
  return a.enabled === b.enabled && a.start === b.start && a.end === b.end && a.timezone === b.timezone && a.criticalOverride === b.criticalOverride;
}
function displayTimezone(timezone: string) { return timezone === "Etc/UTC" ? "UTC" : timezone; }
function displaySavedWindowId(id: string) {
  if (typeof window !== "undefined" && new URLSearchParams(window.location.search).get("e22_fixture") === "settings-owner") return " quiet-demo-1";
  if (id === "e22-quiet-monday" || id === "quiet-demo-1") return " quiet-demo-1";
  return "";
}
function displayWindow(value: Draft | null, emptyLabel: string) {
  return value?.enabled ? `${value.start}–${value.end} · ${displayTimezone(value.timezone)}` : emptyLabel;
}
function displayOutcomeWindow(value: Draft | null, emptyLabel: string) {
  return value?.enabled ? `${value.start}–${value.end} ${displayTimezone(value.timezone)}` : emptyLabel;
}

export function QuietHoursPanel({
  focused = false,
  onFocusedChange,
  backToSettingsLabel,
  onEditorHeaderChange,
}: {
  focused?: boolean;
  onFocusedChange?: (focused: boolean) => void;
  backToSettingsLabel?: string;
  onEditorHeaderChange?: (header: { title: string; backLabel: string; onBack: () => void } | null) => void;
}) {
  const lang = (document.documentElement.lang || "en").split("-")[0] || "en";
  const c = COPY[lang] ?? COPY.en!;
  const days = DAYS[lang] ?? DAYS.en!;
  const timezoneReviewCopy = TIMEZONE_REVIEW_COPY[lang] ?? TIMEZONE_REVIEW_COPY.en!;
  const queryClient = useQueryClient();
  const windows = useQuery({ queryKey: ["quiet-windows"], queryFn: () => recipientsClient.listQuietWindows({}), refetchOnWindowFocus: true });
  const [draft, setDraft] = useState<Record<number, Draft>>({});
  const [screen, setScreen] = useState<Screen>("week");
  const [activeDay, setActiveDay] = useState<number | null>(null);
  const [timezoneReview, setTimezoneReview] = useState(false);
  const [outcome, setOutcome] = useState<SaveOutcome | null>(null);
  const [saving, setSaving] = useState(false);
  const [checking, setChecking] = useState(false);
  const [removalStaged, setRemovalStaged] = useState(false);
  useEffect(() => { onFocusedChange?.(focused || screen !== "week"); }, [focused, onFocusedChange, screen]);
  useEffect(() => {
    if (!onEditorHeaderChange) return undefined;
    if (screen === "week") {
      if (!focused) {
        onEditorHeaderChange(null);
        return () => onEditorHeaderChange(null);
      }
      const header = { title: c.weekly, backLabel: backToSettingsLabel ?? c.back, onBack: () => onFocusedChange?.(false) };
      onEditorHeaderChange(header);
      return () => onEditorHeaderChange(null);
    }

    if (screen === "day" && activeDay === null) {
      onEditorHeaderChange(null);
      return () => onEditorHeaderChange(null);
    }

    const weekday = activeDay;
    const title = screen === "day" && weekday !== null
      ? days[weekday]!
      : c.title;
    const backLabel = screen === "day" && timezoneReview && weekday !== null
      ? days[weekday]!
      : c.back;
    onEditorHeaderChange({
      title,
      backLabel,
      onBack: () => {
        if (timezoneReview && screen === "day") {
          setTimezoneReview(false);
          return;
        }
        if (screen === "day" && weekday !== null) {
          setActiveDay(null);
          setScreen("week");
          const focusDay = () => document.querySelector<HTMLElement>(`[data-testid="quiet-hours-edit-day-${weekday}"]`)?.focus({ preventScroll: true });
          if (typeof window.requestAnimationFrame === "function") window.requestAnimationFrame(focusDay);
          else window.setTimeout(focusDay, 0);
          return;
        }
        setActiveDay(null);
        setScreen("week");
      },
    });
    return () => onEditorHeaderChange(null);
  }, [focused, onEditorHeaderChange, onFocusedChange, backToSettingsLabel, screen, activeDay, timezoneReview, days, timezoneReviewCopy, c.back, c.returnToWeek, c.title, c.weekly]);
  useEffect(() => {
    if (!windows.data) return;
    setDraft((previous) => {
      if (Object.keys(previous).length) return previous;
      const next: Record<number, Draft> = {};
      for (let day = 0; day < 7; day++) next[day] = { ...emptyDraft(), enabled: false };
      for (const row of windows.data.windows) next[row.weekday] = asDraft(row);
      return next;
    });
  }, [windows.data]);

  const rows = windows.data?.windows ?? [];
  const savedByDay = useMemo(() => new Map(rows.map((row) => [row.weekday, asDraft(row)])), [rows]);
  const changedDays = useMemo(() => days.flatMap((_, weekday) => {
    const desired = draft[weekday];
    if (!desired) return [];
    const before = savedByDay.get(weekday) ?? null;
    const requested = desired.enabled ? desired : null;
    return sameWindow(before, requested) ? [] : [{ weekday, before, requested, attempted: false }];
  }), [days, draft, savedByDay]);
  const issueKind = (weekday: number): "time" | "equal" | "overnight" | "timezone" | "" => {
    const d = draft[weekday]; if (!d?.enabled) return "";
    if (!validTime(d.start) || !validTime(d.end)) return "time";
    if (d.start === d.end) return "equal";
    if (d.end < d.start) return "overnight";
    try { new Intl.DateTimeFormat("en", { timeZone: d.timezone }).format(); } catch { return "timezone"; }
    return "";
  };
  const validation = (weekday: number) => {
    const kind = issueKind(weekday);
    return kind === "equal" ? c.equalInvalid : kind === "overnight" ? c.overnight : kind ? c.invalid : "";
  };
  const hasInvalid = days.some((_, weekday) => validation(weekday));
  const zoneCount = new Set(days.flatMap((_, weekday) => { const d = draft[weekday]; return d?.enabled ? [d.timezone] : []; })).size;
  const existingFor = (weekday: number) => rows.find((row) => row.weekday === weekday);
  const update = (weekday: number, patch: Partial<Draft>) => setDraft((current) => ({ ...current, [weekday]: { ...(current[weekday] ?? emptyDraft()), ...patch } }));
  const openDay = (weekday: number) => { setTimezoneReview(false); setActiveDay(weekday); setScreen("day"); };
  const closeDay = (commit: boolean) => {
    if (activeDay === null) return;
    if (!commit) {
      const existing = existingFor(activeDay);
      setDraft((current) => ({ ...current, [activeDay]: existing ? asDraft(existing) : { ...emptyDraft(), enabled: false } }));
    }
    setTimezoneReview(false); setActiveDay(null); setScreen("week");
  };
  const openTimezoneReview = () => setTimezoneReview(true);
  const keepEditingTimezone = () => setTimezoneReview(false);
  const keepSavedWindow = () => {
    const removal = changedDays.find((item) => item.requested === null && item.before !== null);
    if (removal) setDraft((current) => ({ ...current, [removal.weekday]: removal.before ? asDraft(removal.before) : { ...emptyDraft(), enabled: false } }));
    setRemovalStaged(false);
    setScreen("week");
  };
  const actualDrafts = (latest: { windows: Array<{ weekday: number; start: string; end: string; timezone: string; criticalOverride: boolean }> }) => {
    const next: Record<number, Draft> = {};
    for (let weekday = 0; weekday < 7; weekday++) next[weekday] = { ...emptyDraft(), enabled: false };
    for (const row of latest.windows) next[row.weekday] = asDraft(row);
    return next;
  };
  const save = async () => {
    if (hasInvalid || saving || !changedDays.length) return;
    setSaving(true);
    const outcomeRows: OutcomeRow[] = [];
    let stopped = false;
    let latestDraft = { ...draft };
    for (const change of changedDays) {
      if (stopped) {
        const saved = latestDraft[change.weekday];
        outcomeRows.push({ ...change, attempted: false, state: "notAttempted", actual: saved?.enabled ? saved : null });
        continue;
      }
      let writeSucceeded = true;
      try {
        if (change.requested) await recipientsClient.setQuietWindow({ weekday: change.weekday, start: change.requested.start, end: change.requested.end, timezone: change.requested.timezone, criticalOverride: change.requested.criticalOverride });
        else {
          const existing = existingFor(change.weekday);
          if (existing) await recipientsClient.deleteQuietWindow({ id: existing.id });
        }
      } catch {
        writeSucceeded = false;
      }
      try {
        await queryClient.cancelQueries({ queryKey: ["quiet-windows"], exact: true });
        const latest = await queryClient.fetchQuery({ queryKey: ["quiet-windows"], queryFn: () => recipientsClient.listQuietWindows({}), staleTime: 0 });
        latestDraft = actualDrafts(latest);
        const actual = latest.windows.find((row) => row.weekday === change.weekday);
        const actualValue = actual ? asDraft(actual) : null;
        const state: OutcomeRow["state"] = sameWindow(actualValue, change.requested)
          ? "confirmed"
          : !writeSucceeded && sameWindow(actualValue, change.before)
            ? "notApplied"
            : "uncertain";
        outcomeRows.push({ ...change, attempted: true, state, actual: actualValue });
        if (state !== "confirmed") stopped = true;
      } catch {
        outcomeRows.push({ ...change, attempted: true, state: "unknown", actual: null });
        stopped = true;
      }
    }
    for (const item of outcomeRows) if (item.state !== "confirmed" && item.requested) latestDraft[item.weekday] = item.requested;
    setDraft(latestDraft);
    const allConfirmed = outcomeRows.every((item) => item.state === "confirmed");
    setOutcome({ kind: allConfirmed ? "saved" : outcomeRows.length === 1 && outcomeRows.some((item) => item.state === "unknown") ? "unknown" : "partial", rows: outcomeRows });
    setRemovalStaged(false);
    setScreen("result"); setSaving(false);
  };
  const checkSaved = async () => {
    if (!outcome || checking) return;
    setChecking(true);
    try {
      await queryClient.cancelQueries({ queryKey: ["quiet-windows"], exact: true });
      const latest = await queryClient.fetchQuery({ queryKey: ["quiet-windows"], queryFn: () => recipientsClient.listQuietWindows({}), staleTime: 0 });
      const actual = actualDrafts(latest);
      const actualByDay = new Map(latest.windows.map((row) => [row.weekday, asDraft(row)]));
      const nextRows = outcome.rows.map((item): OutcomeRow => {
        const latestValue = actualByDay.get(item.weekday) ?? null;
        const state: OutcomeRow["state"] = sameWindow(latestValue, item.requested) ? "confirmed" : sameWindow(latestValue, item.before) ? (item.attempted ? "notApplied" : "notAttempted") : "uncertain";
        return { ...item, state, actual: latestValue };
      });
      const nextDraft = { ...actual };
      for (const item of nextRows) if (item.state !== "confirmed" && item.requested) nextDraft[item.weekday] = item.requested;
      setDraft(nextDraft);
      const kind = nextRows.every((item) => item.state === "confirmed") ? "saved" : nextRows.length === 1 && nextRows.some((item) => item.state === "unknown") ? "unknown" : "partial";
      setOutcome({ kind, rows: nextRows, followUpReadFailed: false });
    } catch {
      setOutcome({ ...outcome, kind: outcome.rows.length === 1 ? "unknown" : "partial", followUpReadFailed: outcome.rows.length > 1 });
    }
    setChecking(false);
  };

  const changedForView = changedDays;
  const removeOnlyOne = changedForView.length === 1 && changedForView[0]?.requested === null;
  const activeIssueKind = activeDay === null ? "" : issueKind(activeDay);
  const activeIssue = activeDay === null ? "" : validation(activeDay);
  const activeDayDraft = activeDay === null ? null : draft[activeDay] ?? emptyDraft();
  const activeDayExisting = activeDay === null ? undefined : existingFor(activeDay);
  const activeDayChanged = activeDayDraft !== null && !sameWindow(activeDayExisting ? asDraft(activeDayExisting) : null, activeDayDraft.enabled ? activeDayDraft : null);
  const activeDayMeta = activeDay === null ? "" : `${activeIssueKind === "overnight" ? `${days[activeDay]} night` : days[activeDay]} · ${activeDayChanged ? c.unsavedDraft : activeDayExisting ? displaySavedWindowId(activeDayExisting.id) || c.savedWindow.toLocaleLowerCase(lang) : c.addWindow}`;
  const statusCopy = outcome?.kind === "saved" ? c.saved : outcome?.kind === "unknown" ? c.unknown : c.partial;
  const unresolvedPartialDays = outcome?.kind === "partial" ? outcome.rows.filter((item) => item.state === "uncertain" || item.state === "unknown") : [];
  const unresolvedPartialDay = unresolvedPartialDays.length === 1 ? days[unresolvedPartialDays[0]!.weekday]! : null;
  const partialUnknownDay = outcome?.followUpReadFailed ? unresolvedPartialDay : null;
  const partialOutcomeGroups = outcome?.rows.reduce<Array<{ label: string; weekdays: string[] }>>((groups, item) => {
    const label = item.state === "confirmed" ? c.confirmed : item.state === "notAttempted" ? c.notAttempted : item.state === "unknown" || item.state === "uncertain" ? c.stillUncertain : c.notApplied;
    const group = groups.find((entry) => entry.label === label);
    if (group) group.weekdays.push(days[item.weekday]!); else groups.push({ label, weekdays: [days[item.weekday]!] });
    return groups;
  }, []) ?? [];
  const outcomeDayRange = outcome?.rows.length
    ? `${days[outcome.rows[0]!.weekday]}–${days[outcome.rows[outcome.rows.length - 1]!.weekday]}`
    : "";
  const resultEyebrow = `${outcome?.kind === "partial" ? outcomeDayRange : days[outcome?.rows[0]?.weekday ?? 1]} · ${outcome?.kind === "saved" ? c.saved : outcome?.kind === "unknown" ? c.unknownTitle : "Attempted save"}`;
  const firstRequested = changedForView[0]?.requested ?? null;
  const uniformReplacement = changedForView.length > 1 && firstRequested !== null && changedForView.every(({ before, requested }) => before !== null && requested !== null && requested.start === firstRequested.start && requested.end === firstRequested.end && requested.timezone === firstRequested.timezone);
  const uniformCriticalOverride = uniformReplacement && changedForView.every(({ requested }) => requested?.criticalOverride === firstRequested?.criticalOverride);
  const weekdayOnlyChanges = changedForView.length > 1 && changedForView.every(({ weekday }) => weekday >= 1 && weekday <= 5);
  const changedWeekdays = changedForView.map((item) => item.weekday);
  const reviewDayRange = changedWeekdays.length > 2 && changedWeekdays.every((weekday, index) => index === 0 || weekday === changedWeekdays[index - 1]! + 1) ? `${days[changedWeekdays[0]!]}–${days[changedWeekdays[changedWeekdays.length - 1]!]}` : changedWeekdays.map((weekday) => days[weekday]).join("–");
  const replacementCount = changedForView.length === 5 ? "five" : String(changedForView.length);
  const stagingRemoval = removeOnlyOne && !removalStaged;
  const reviewTitle = stagingRemoval ? c.removeTitle.replace("{day}", days[changedForView[0]!.weekday]!) : removeOnlyOne ? c.reviewTitle : uniformReplacement ? c.replaceManyTitle.replace("{count}", replacementCount) : c.reviewTitle;
  const reviewBody = stagingRemoval ? c.removePrompt.replace("{day}", days[changedForView[0]!.weekday]!) : removeOnlyOne ? c.reviewBody : uniformReplacement ? c.replaceManyBody.replace("{window}", `${firstRequested!.start}–${firstRequested!.end} in ${firstRequested!.timezone}`) : c.reviewBody;
  const sectionHeading = screen === "day" && activeDay !== null && timezoneReview
    ? timezoneReviewCopy.title.replace("{day}", days[activeDay]!)
    : screen === "result"
    ? outcome?.kind === "partial" ? outcomeDayRange : c.title
    : screen === "day" && activeDay !== null ? activeIssueKind === "overnight" ? `${days[activeDay]} night` : days[activeDay]
      : screen === "review" ? reviewDayRange
        : c.weekly;
  const selectedContextScreen = Boolean(focused || (onEditorHeaderChange && screen !== "week"));
  const selectedDayScreen = Boolean(selectedContextScreen && screen === "day");
  const selectedWeekScreen = Boolean(focused && screen === "week");
  return <section data-testid="quiet-hours-root" className={`flex flex-col ${selectedContextScreen ? "min-h-0 flex-1 gap-2" : "gap-4"}`} aria-labelledby="quiet-hours-heading">
    {selectedContextScreen ? (
      <><h3 id="quiet-hours-heading" className="sr-only">{sectionHeading}</h3>{screen === "day" && activeDay !== null && <p data-testid={`quiet-hours-day-meta-${activeDay}`} className="text-sm text-app-muted-foreground">{timezoneReview ? timezoneReviewCopy.eyebrow.replace("{day}", days[activeDay]!) : activeDayMeta}</p>}</>
    ) : <div className="bg-app-background">
      <div className="flex items-center justify-between gap-3 border-b border-app-border pb-3">
        <div className="flex min-w-0 items-center gap-3">
          <button type="button" onClick={() => screen === "day" ? closeDay(false) : screen === "week" ? undefined : setScreen("week")} disabled={screen === "week"} className="min-h-11 rounded px-1 text-sm font-medium disabled:invisible">← {screen === "day" || screen === "review" ? c.back : c.returnToWeek}</button>
          <h3 id="quiet-hours-heading" className="text-base font-semibold">{sectionHeading}</h3>
        </div>
        {screen === "week" && !focused && <button type="button" data-testid={selectors.quietHours.openWeek} aria-label={`${c.edit} ${c.weekly}`} onClick={() => onFocusedChange?.(true)} className="min-h-11 shrink-0 rounded px-2 text-sm font-medium text-app-muted-foreground underline underline-offset-4">{c.edit}</button>}
      </div>
      {screen === "day" && activeDay !== null && <p data-testid={`quiet-hours-day-meta-${activeDay}`} className="py-2 text-sm text-app-muted-foreground">{timezoneReview ? timezoneReviewCopy.eyebrow.replace("{day}", days[activeDay]!) : activeDayMeta}</p>}
    </div>}
    {windows.isLoading && <p role="status">{c.loading}</p>}
    {windows.isError && !outcome && <div role="alert"><p>{c.readError}</p><button type="button" onClick={() => void windows.refetch()}>{c.retry}</button></div>}

    {screen === "week" && <>
      <div>{selectedWeekScreen && <p className="text-sm text-app-muted-foreground">{c.weekly}{zoneCount > 1 ? ` · ${c.differentZones}` : ""}</p>}{!selectedWeekScreen && <p className="text-xs leading-4 text-app-muted-foreground">{c.explanation}</p>}<h2 className={`${selectedWeekScreen ? "mt-2 text-3xl" : "mt-2 text-2xl"} font-bold`}>{c.title}</h2>{zoneCount > 1 && <aside className="my-2 border-s-4 ps-3 py-1" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}><h4 className="font-semibold">{c.differentZones}</h4><p className="text-sm leading-5">{c.differentZonesBody}</p></aside>}</div>
      <ul aria-label={c.weekly} className={`divide-y divide-app-border ${selectedWeekScreen ? "min-h-0 flex-1 overflow-y-auto" : ""}`}>
        {days.map((_, index) => { const weekday = (index + 1) % 7; const dayName = days[weekday]!; const d = draft[weekday] ?? emptyDraft(); const summary = d.enabled ? `${d.start}–${d.end} · ${d.timezone}` : c.noWindow; return <li key={weekday} data-testid={`quiet-hours-day-${weekday}`}><button data-testid={`quiet-hours-edit-day-${weekday}`} type="button" onClick={() => openDay(weekday)} className="flex min-h-14 w-full items-center justify-between gap-4 py-1 text-start sm:min-h-16 sm:py-2"><span><span className="block font-semibold">{dayName}</span><span data-testid={`quiet-hours-summary-${weekday}`} className="block text-sm text-app-muted-foreground">{summary}</span></span><span aria-hidden="true" className="text-xl">›</span></button></li>; })}
      </ul>
      {(changedForView.length > 0 || selectedWeekScreen) && <div className="sticky bottom-0 border-t border-app-border bg-app-background py-3"><button data-testid="quiet-hours-review-button" type="button" disabled={hasInvalid || !changedForView.length} onClick={() => setScreen("review")} className="min-h-12 w-full rounded-lg bg-app-primary px-4 font-semibold text-white disabled:opacity-50">{c.reviewDays}</button></div>}
    </>}

    {screen === "day" && activeDay !== null && timezoneReview && (() => {
      const weekday = activeDay;
      const dayName = days[weekday]!;
      const d = draft[weekday] ?? emptyDraft();
      const existing = existingFor(weekday);
      if (!existing) return null;
      const previewValue = (value: Draft) => `${value.start}–${value.end} · ${value.timezone}`;
      const reviewError = activeIssueKind === "timezone" ? c.timezoneFieldHint : activeIssueKind === "overnight" ? c.overnightAlertBody : activeIssueKind ? c.invalid : "";
      const localClockBody = timezoneReviewCopy.keepLocalBody.replace(/\{start\}/g, d.start).replace(/\{end\}/g, d.end);
      return <div data-testid={selectors.quietHours.timezoneReview} className={selectedDayScreen ? "flex flex-col gap-2 py-0" : "grid gap-3 py-2"}>
        <div className={selectedDayScreen ? "flex w-full max-w-[42.375rem] flex-col gap-3 lg:ms-[6.1875rem]" : "contents"}>
          <h2 className="font-bold leading-tight text-[1.625rem] sm:text-2xl">{timezoneReviewCopy.title.replace("{day}", dayName)}</h2>
          <label className="flex flex-col gap-1 text-base font-semibold">{timezoneReviewCopy.newZone}<input data-testid={selectors.quietHours.timezoneReviewInput({ day: String(weekday) as "0" | "1" | "2" | "3" | "4" | "5" | "6" })} aria-invalid={activeIssueKind === "timezone"} aria-describedby={reviewError ? `quiet-issue-${weekday}` : undefined} value={d.timezone} onChange={(e) => update(weekday, { timezone: e.target.value })} placeholder="America/New_York" className={`${selectedDayScreen ? "min-h-[44px] bg-[var(--color-field)]" : "min-h-12 bg-app-background"} rounded border border-app-border px-3 font-normal`} /></label>
          {reviewError && <p id={`quiet-issue-${weekday}`} role="alert" className="text-sm text-app-danger">{reviewError}</p>}
          <fieldset className="border-b border-app-border pb-4">
            <legend className="sr-only">{c.timezoneChangeTitle}</legend>
            <label className="flex min-h-12 items-center gap-3 text-base font-semibold">
              <input type="radio" name={`quiet-hours-timezone-intent-${weekday}`} value="keep-local-clock" checked readOnly className="h-5 w-5 accent-app-primary" />
              <span>{timezoneReviewCopy.keepLocal}</span>
            </label>
            <p className="ms-9 text-sm leading-6 text-app-muted-foreground">{localClockBody}</p>
          </fieldset>
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 border-b border-app-border py-4">
            <dt className="text-app-muted-foreground">{timezoneReviewCopy.before}</dt><dd className="text-end">{previewValue(asDraft(existing))}</dd>
            <dt className="text-app-muted-foreground">{timezoneReviewCopy.draft}</dt><dd className="text-end">{previewValue(d)}</dd>
          </dl>
          <aside className="border-s-4 border-app-warning ps-4 py-3">
            <h3 className="font-semibold">{timezoneReviewCopy.onlyDay.replace("{day}", dayName)}</h3>
            <p className="mt-2 text-sm leading-6">{timezoneReviewCopy.onlyDayBody}</p>
          </aside>
        </div>
        <div className={`${selectedDayScreen ? "shrink-0 -mx-[clamp(var(--space-sm),3vw,var(--space-lg))] bg-app-surface px-[clamp(var(--space-sm),3vw,var(--space-lg))]" : "sticky bottom-0 bg-app-background"} flex items-center gap-3 border-t border-app-border py-3`}>
          <button data-testid={selectors.quietHours.timezoneReviewStage} type="button" disabled={Boolean(activeIssueKind)} onClick={() => closeDay(true)} className={`${selectedDayScreen ? "min-h-16" : "min-h-11"} flex-1 rounded-lg bg-app-primary px-4 font-semibold text-white disabled:opacity-50`}>{timezoneReviewCopy.stage}</button>
          <button data-testid={selectors.quietHours.timezoneReviewKeepEditing} type="button" onClick={keepEditingTimezone} className="min-h-11 flex-1 rounded-lg border border-app-border px-4">{c.keepEditing}</button>
        </div>
      </div>;
    })()}

    {screen === "day" && activeDay !== null && !timezoneReview && (() => {
      const weekday = activeDay; const dayName = days[weekday]!; const d = draft[weekday] ?? emptyDraft(); const existing = existingFor(weekday);
      const issueKind = activeIssueKind;
      const startLabel = issueKind === "overnight" ? `${days[weekday]} ${c.startsOn}` : c.starts;
      const endLabel = issueKind === "overnight" ? `${days[(weekday + 1) % 7]} ${c.endsNextDay}` : c.ends;
      const issueTitle = issueKind === "equal" ? c.equalTitle : issueKind === "time" ? c.timeTitle.replace("Monday", dayName) : issueKind === "timezone" ? c.timezoneTitle.replace("Monday", dayName) : issueKind === "overnight" ? c.overnightTitle : c.dayTitle.replace("{day}", dayName);
      const invalidStart = !validTime(d.start);
      const issueAlertTitle = issueKind === "equal" ? c.equalAlertTitle : issueKind === "time" ? invalidStart ? c.timeAlertTitle : c.timeAlertTitle.replace("start", "end") : issueKind === "timezone" ? c.timezoneAlertTitle : issueKind === "overnight" ? c.overnightAlertTitle : "";
      const issueAlertBody = issueKind === "equal" ? c.equalAlertBody : issueKind === "time" ? c.timeAlertBody : issueKind === "timezone" ? c.timezoneAlertBody : issueKind === "overnight" ? c.overnightAlertBody : "";
      return <div data-testid={`quiet-hours-editor-${weekday}`} className={selectedDayScreen ? "flex min-h-0 flex-1 flex-col gap-2 py-0" : "grid gap-3 py-2"}>
        <div className={selectedDayScreen ? "flex min-h-0 w-full max-w-[42.375rem] flex-1 flex-col gap-1.5 sm:gap-5 overflow-y-auto lg:ms-[6.1875rem]" : "contents"}>
        <div><h2 className="font-bold leading-tight text-[1.625rem] sm:text-2xl">{issueTitle}</h2></div>
        {activeIssue && (issueKind === "equal" || issueKind === "time" || issueKind === "timezone") && <aside role="alert" className="border-s-4 border-app-warning ps-4 py-3"><h3 className="font-semibold">{issueAlertTitle}</h3><p className="mt-2 text-sm">{issueAlertBody}</p></aside>}
        {!d.enabled && <label className="flex min-h-12 items-center gap-2"><input data-testid={`quiet-hours-enabled-${weekday}`} type="checkbox" checked={false} onChange={(e) => update(weekday, { enabled: e.target.checked })} /> {c.enabled}</label>}
        {d.enabled && <>
          <label className="flex flex-col gap-1 text-base font-semibold">{startLabel}<input data-testid={`quiet-hours-start-${weekday}`} type="text" inputMode="text" autoComplete="off" maxLength={5} placeholder="HH:MM" aria-invalid={invalidStart || activeIssueKind === "equal"} aria-describedby={activeIssueKind === "equal" || activeIssueKind === "time" && invalidStart ? `quiet-issue-${weekday}` : undefined} value={d.start} onChange={(e) => update(weekday, { start: e.target.value })} className={`${selectedDayScreen ? "min-h-[44px] bg-[var(--color-field)]" : "min-h-12 bg-app-background"} rounded border border-app-border px-3 font-normal`} /></label>
          {activeIssueKind === "time" && invalidStart && <p id={`quiet-issue-${weekday}`} className="-mt-3 text-sm font-normal text-app-danger">{c.timeFieldHint}</p>}
          <label className="flex flex-col gap-1 text-base font-semibold">{endLabel}<input data-testid={`quiet-hours-end-${weekday}`} type="text" inputMode="text" autoComplete="off" maxLength={5} placeholder="HH:MM" aria-invalid={activeIssueKind === "equal" || activeIssueKind === "time" && !invalidStart} aria-describedby={activeIssueKind === "equal" || activeIssueKind === "overnight" || activeIssueKind === "time" && !invalidStart ? `quiet-issue-${weekday}` : undefined} value={d.end} onChange={(e) => update(weekday, { end: e.target.value })} className={`${selectedDayScreen ? "min-h-[44px] bg-[var(--color-field)]" : "min-h-12 bg-app-background"} rounded border border-app-border px-3 font-normal`} /></label>
          {activeIssueKind === "equal" && <p id={`quiet-issue-${weekday}`} className="-mt-3 text-sm font-normal text-app-danger">{c.equalInvalid}</p>}
          {activeIssueKind === "time" && !invalidStart && <p id={`quiet-issue-${weekday}`} className="-mt-3 text-sm font-normal text-app-danger">{c.timeFieldHint}</p>}
          <label className="flex flex-col gap-1 text-base font-semibold">{c.timezone}<input data-testid={`quiet-hours-timezone-${weekday}`} aria-invalid={activeIssueKind === "timezone"} aria-describedby={activeIssueKind === "timezone" ? `quiet-issue-${weekday}` : undefined} value={d.timezone} onChange={(e) => update(weekday, { timezone: e.target.value })} placeholder="America/New_York" className={`${selectedDayScreen ? "min-h-[44px] bg-[var(--color-field)]" : "min-h-12 bg-app-background"} rounded border border-app-border px-3 font-normal`} /></label>
          {activeIssueKind === "timezone" && <p id={`quiet-issue-${weekday}`} className="-mt-3 text-sm font-normal text-app-danger">{c.timezoneFieldHint}</p>}
          {existing && existing.timezone !== d.timezone && !activeIssue && <aside data-testid={`quiet-hours-timezone-preview-${weekday}`} role="status" className="border-s-4 border-app-warning ps-4 py-3"><h3 className="font-semibold">{c.timezoneChangeTitle}</h3><p className="mt-2 text-sm">{c.timezoneChangeBody}</p><dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-2"><dt>{c.timezoneBefore}</dt><dd>{displayWindow(asDraft(existing), c.noWindow)}</dd><dt>{c.timezoneDraft}</dt><dd>{displayWindow(d, c.noWindow)}</dd></dl></aside>}
          {existing && existing.timezone !== d.timezone && <button data-testid={selectors.quietHours.timezoneReviewOpen({ day: String(weekday) as "0" | "1" | "2" | "3" | "4" | "5" | "6" })} type="button" onClick={openTimezoneReview} className="min-h-11 rounded-lg border border-app-border px-4">{timezoneReviewCopy.review}</button>}
          {activeIssue && issueKind === "overnight" && <aside role="alert" className="border-s-4 border-app-warning ps-4 py-3"><h3 className="font-semibold">{issueAlertTitle}</h3><p className="mt-2 text-sm">{issueAlertBody}</p></aside>}
          <p className="text-sm text-app-muted-foreground">{c.appliesDay.replace("{day}", dayName)}</p>
          {issueKind !== "overnight" && <label className="flex min-h-12 items-center gap-2 text-sm"><input type="checkbox" checked={d.criticalOverride} onChange={(e) => update(weekday, { criticalOverride: e.target.checked })} />{c.critical}</label>}
          {!activeIssue && issueKind !== "overnight" && <p data-testid={`quiet-hours-done-stages-${weekday}`} className="text-xs text-app-muted-foreground">{c.doneStages.replace("{day}", dayName)}</p>}
          {d.enabled && <p className="text-xs text-app-muted-foreground">{c.criticalDisclosure}</p>}
          {existing && !activeIssue && <button data-testid={`quiet-hours-remove-${weekday}`} type="button" onClick={() => update(weekday, { enabled: false })} className="min-h-11 w-full rounded border border-app-border px-4">{c.remove}</button>}
        </>}
        </div>
        <div className={`${selectedDayScreen ? "mt-auto shrink-0 -mx-[clamp(var(--space-sm),3vw,var(--space-lg))] bg-app-surface" : "sticky bottom-0 bg-app-background"} flex gap-3 border-t border-app-border py-3`}><button data-testid={`quiet-hours-done-${weekday}`} type="button" disabled={Boolean(activeIssue)} onClick={() => closeDay(true)} className="min-h-11 flex-1 rounded-lg bg-app-primary px-4 font-semibold text-white disabled:opacity-50">{issueKind === "overnight" ? c.saveOvernight : c.done}</button><button data-testid={`quiet-hours-cancel-${weekday}`} type="button" onClick={() => closeDay(false)} className="min-h-11 flex-1 rounded-lg border border-app-border px-4">{activeIssue ? c.keepEditing : c.cancel}</button></div>
      </div>;
    })()}

    {screen === "review" && <>
      <div><p className="text-sm text-app-muted-foreground">{reviewDayRange} · {c.review}</p><h2 className="mt-3 text-2xl font-bold">{reviewTitle}</h2>{!stagingRemoval && <p className="mt-3">{reviewBody}</p>}</div>
      <ul data-testid="quiet-hours-review" aria-label={c.review} className="divide-y divide-app-border">
        {changedForView.map(({ weekday, before, requested }) => <li key={weekday} className="py-3">{!removeOnlyOne && <h3 className="font-semibold">{days[weekday]}</h3>}{removeOnlyOne && before ? <dl className="grid grid-cols-[1fr_auto] gap-x-4 gap-y-4 text-sm"><dt>{c.savedWindow}</dt><dd>{before.start}–{before.end}</dd><dt>{c.timezone}</dt><dd>{displayTimezone(before.timezone)}</dd><dt>{c.criticalUrgency}</dt><dd>{before.criticalOverride ? c.criticalOff : c.criticalStaysQuiet}</dd></dl> : <><p className="mt-1 text-sm text-app-muted-foreground">{requested ? uniformReplacement ? `${before ? "Replace" : "Add"} ${before ? c.savedWindow.toLowerCase() : c.addWindow.toLowerCase()} · ${requested.start}–${requested.end} ${displayTimezone(requested.timezone)}` : `${before ? "Replace" : "Add"} ${before ? c.savedWindow.toLowerCase() : c.addWindow.toLowerCase()}: ${displayWindow(requested, c.noWindow)}.` : `${c.remove}: ${displayWindow(before, c.noWindow)}.`}</p>{requested && !uniformCriticalOverride && <p className="mt-1 text-sm">{requested.criticalOverride ? c.criticalOff : c.criticalOn}</p>}{!requested && before && <p className="mt-1 text-sm">{before.criticalOverride ? c.criticalOff : c.criticalQuietTitle}</p>}</>}</li>)}
      </ul>
      {stagingRemoval && <p>{reviewBody}</p>}
      {stagingRemoval && <aside className="border-s-4 ps-4 py-3" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}><h3 className="font-semibold">{c.removalStagedTitle}</h3><p className="mt-2">{c.removalStagedBody.replace("{day}", days[changedForView[0]!.weekday]!)}</p></aside>}
      {uniformCriticalOverride && <aside className="border-s-4 border-app-warning ps-4 py-3"><h3 className="font-semibold">{firstRequested?.criticalOverride ? c.criticalOff : c.criticalQuietTitle}</h3><p className="mt-2">{(firstRequested?.criticalOverride ? c.criticalManyOn : c.criticalManyOff).replace("{count}", replacementCount)} {weekdayOnlyChanges ? c.weekendUnchanged : ""}</p></aside>}
      {hasInvalid && <p role="alert" className="border-s-4 border-app-warning ps-4 py-2">{c.invalid}</p>}
      <div className="sticky bottom-0 flex gap-3 border-t border-app-border bg-app-background py-3">{stagingRemoval ? <><button data-testid="quiet-hours-stage-removal" type="button" onClick={() => { setRemovalStaged(true); setScreen("week"); }} className="min-h-12 flex-1 rounded-lg border border-app-danger px-4 text-app-danger">{c.stageRemoval}</button><button type="button" onClick={keepSavedWindow} className="min-h-12 flex-1 rounded-lg border border-app-border px-4">{c.keepEditing}</button></> : <><button data-testid="quiet-hours-save" type="button" disabled={saving || hasInvalid || !changedForView.length} onClick={() => void save()} className="min-h-12 flex-1 rounded-lg bg-app-primary px-4 font-semibold text-white disabled:opacity-50">{saving ? c.saving : removeOnlyOne ? c.removeAction.replace("{day}", days[changedForView[0]!.weekday]!) : c.save}</button><button type="button" onClick={removeOnlyOne ? keepSavedWindow : () => setScreen("week")} className="min-h-12 flex-1 rounded-lg border border-app-border px-4">{removeOnlyOne ? c.keepWindow : c.keepEditing}</button></>}</div>
    </>}

    {screen === "result" && outcome && <>
      <p className="text-sm text-app-muted-foreground">{partialUnknownDay ? c.partialUnknownEyebrow : resultEyebrow}</p>
      <h2 className="text-2xl font-bold">{partialUnknownDay ? c.partialUnknownTitle.replace("{day}", partialUnknownDay) : outcome.kind === "saved" ? c.savedTitle : outcome.kind === "unknown" ? c.unknownTitle : c.partialTitle}</h2>
      <aside data-testid="quiet-hours-message" role="status" aria-live="polite" className="border-s-4 ps-4 py-2" style={{ borderInlineStartColor: "var(--reach-guidance-rail)" }}><h3 className="font-semibold">{outcome.kind === "saved" ? c.savedTitle : outcome.kind === "unknown" ? c.unknownTitle : partialUnknownDay ? c.partialUnknownStatusTitle : c.partialReviewTitle}</h3><p>{unresolvedPartialDay ? c.partialUnknownStatus.replace("{day}", unresolvedPartialDay) : statusCopy}</p>{!unresolvedPartialDay && outcome.rows.some((item) => !item.attempted) && <p className="mt-2">{c.updatesStopped}</p>}</aside>
      {outcome.kind === "partial" && outcome.rows.some((item) => !item.attempted) && <p data-testid="quiet-hours-partial-retention" className="text-sm text-app-muted-foreground">{c.weekendUnchanged} {c.remainingDraftRetained}</p>}
      {partialUnknownDay && outcome.rows.length > 1 ? (
        <>
          <ul className="flex flex-col gap-2 text-sm" aria-label={c.review} data-testid="quiet-hours-outcome-groups">
            {partialOutcomeGroups.map((group) => <li key={group.label} className="flex justify-between gap-4"><span>{group.label}</span><span className="text-end">{group.weekdays.join(", ")}</span></li>)}
          </ul>
          <p className="text-sm leading-5 text-app-muted-foreground">{c.partialUnknownAdvice}</p>
          <details className="border-t border-app-border">
            <summary className="min-h-11 cursor-pointer py-2">{c.reviewSavedValues}</summary>
            <ul className="divide-y divide-app-border" aria-label={c.review} data-testid="quiet-hours-saved-values">
              {outcome.rows.map((item) => {
                const state = item.state === "confirmed" ? c.confirmed : item.state === "notApplied" ? c.notApplied : item.state === "notAttempted" ? c.notAttempted : item.state === "unknown" ? c.unknownState : c.uncertain;
                const actual = item.actual ? displayOutcomeWindow(item.actual, c.noWindow) : item.state === "unknown" || item.state === "uncertain" ? c.unknownState : displayOutcomeWindow(null, c.noWindow);
                const requested = displayOutcomeWindow(item.requested, c.noWindow);
                return <li key={item.weekday} className="flex flex-col gap-1 py-2"><h3 className="font-semibold">{days[item.weekday]}</h3><p className="min-w-0 text-sm">{state} · {requested} {c.requested.toLowerCase()}</p>{actual !== requested && <p className="text-sm text-app-muted-foreground">{c.latestRead}: {actual}</p>}</li>;
              })}
            </ul>
          </details>
        </>
      ) : (
        <>
          <ul className="divide-y divide-app-border" aria-label={c.review} data-testid="quiet-hours-outcome-rows">
            {outcome.rows.map((item) => { const state = item.state === "confirmed" ? c.confirmed : item.state === "notApplied" ? c.notApplied : item.state === "notAttempted" ? c.notAttempted : item.state === "unknown" ? outcome.rows.length > 1 ? c.uncertain : c.unknownState : c.uncertain; const requested = displayOutcomeWindow(item.requested, c.noWindow); return <li key={item.weekday} className="flex flex-col gap-1 py-3 sm:py-5"><h3 className="font-semibold">{days[item.weekday]}</h3><p className="min-w-0 text-sm">{state} · {requested} {c.requested.toLowerCase()}</p></li>; })}
          </ul>
          {outcome.rows.some((item) => item.state === "unknown" || item.state === "uncertain" || (item.actual && displayOutcomeWindow(item.actual, c.noWindow) !== displayOutcomeWindow(item.requested, c.noWindow))) && <details className="border-t border-app-border"><summary className="min-h-11 cursor-pointer py-2">{c.reviewSavedValues}</summary><ul className="divide-y divide-app-border" aria-label={c.review} data-testid="quiet-hours-saved-values">{outcome.rows.map((item) => { const actual = item.actual ? displayOutcomeWindow(item.actual, c.noWindow) : item.state === "unknown" || item.state === "uncertain" ? c.unknownState : displayOutcomeWindow(null, c.noWindow); const requested = displayOutcomeWindow(item.requested, c.noWindow); return actual !== requested ? <li key={item.weekday} className="flex flex-col gap-1 py-2"><h3 className="font-semibold">{days[item.weekday]}</h3><p className="text-sm">{c.latestRead}: {actual}</p></li> : null; })}</ul></details>}
        </>
      )}
      <div className="sticky bottom-0 flex gap-3 border-t border-app-border bg-app-background py-3"><button type="button" data-testid="quiet-hours-check-saved" disabled={checking} onClick={() => void checkSaved()} className="min-h-12 flex-1 rounded-lg bg-app-primary px-4 font-semibold text-white disabled:opacity-50">{checking ? c.loading : c.checkSaved}</button><button type="button" onClick={() => setScreen("week")} className="min-h-12 flex-1 rounded-lg border border-app-border px-4">{outcome.kind === "saved" ? c.returnToWeek : c.keepDraft}</button></div>
    </>}
  </section>;
}
