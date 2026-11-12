import { useQueryClient } from "@tanstack/react-query";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useTopics } from "../api/activity";
import { apiPathUrl } from "../api/client";
import { useHomeText } from "../api/desk";
import { operatorApi, useDryRun, useReviews, type Review } from "../api/operator";
import { Confirm } from "../components/Confirm";
import { EntryField } from "../components/EntryField";
import { HtmlOpen } from "../components/HtmlOpen";
import { Markdown } from "../components/Markdown";
import { PathText } from "../components/PathText";
import { QueryView } from "../components/QueryView";
import { Section } from "../components/Section";
import { Unreadable } from "../components/Unreadable";
import { When } from "../components/When";
import { WouldWrite } from "../components/WouldWrite";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { entryTooLong } from "../lib/entry";
import { toDate } from "../lib/format";
import { dirname } from "../lib/paths";
import { useDebounced } from "../lib/useDebounced";

/** One review: REVIEW.md and REVIEW.html with their media, proposals, and a verdict per topic. */
export function ReviewPage() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const query = useReviews();
  return (
    <div data-testid={selectors.pages.review} className="rd-page">
      <Link to="/reviews" className="rd-back">{t(strings.reviews.back)}</Link>
      <QueryView query={query}>
        {({ reviews }) => {
          const review = reviews.find((r) => r.id === id);
          if (!review) return <Alert tone="danger" title={t(strings.reviews.missing, { id })} />;
          return <ReviewBody review={review} />;
        }}
      </QueryView>
    </div>
  );
}

function ReviewBody({ review }: { review: Review }) {
  const { t } = useTranslation();
  const [extra, setExtra] = useState<string[]>([]);
  const listed = [...review.topics, ...extra];
  const md = useHomeText(apiPathUrl(review.review?.url ?? ""));
  const html = apiPathUrl(review.reviewHtml?.url ?? "");
  return (
    <>
      <PageHeader headingId="review-heading" title={review.title?.value || review.id} eyebrow={<span className="rd-id">{review.id}</span>} />
      <Unreadable evidence={review.unreadable} />
      <Section id="verdicts" title={t(strings.reviews.verdicts)} note={t(strings.reviews.verdictsNote)}>
        {listed.length === 0 ? <p className="text-body-sm">{t(strings.reviews.noTopics)}</p> : (
          <ul className="rd-rows">
            {listed.map((topic) => (
              <VerdictForm key={topic} review={review} topic={topic} />
            ))}
          </ul>
        )}
        <OtherTopic exclude={listed} onPick={(topic) => setExtra([...extra, topic])} />
      </Section>
      {html ? (
        <Section id="review-html" title={t(strings.reviews.html)}>
          <HtmlOpen src={html} name="REVIEW.html" />
        </Section>
      ) : null}
      {review.review?.url ? (
        <Section id="review-md" title={t(strings.reviews.markdown)}>
          <QueryView query={md}>{(text) => <Markdown content={text} baseDir={review.review?.file ? dirname(review.review.file.path) : review.folder} />}</QueryView>
        </Section>
      ) : null}
      {review.proposals.length > 0 ? (
        <Section id="proposals" title={t(strings.reviews.proposals, { count: review.proposals.length })}>
          <ul className="rd-list">
            {review.proposals.map((p) => (p.file ? <li key={p.file.path}><PathText path={p.file.path} /></li> : null))}
          </ul>
        </Section>
      ) : null}
    </>
  );
}

function VerdictForm({ review, topic }: { review: Review; topic: string }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [text, setText] = useState("");
  const [open, setOpen] = useState(false);
  const recorded = review.verdicts.filter((v) => v.subject === topic);
  const words = useDebounced(text.trim(), 400);
  const preview = useDryRun(["verdict", review.id, topic, words], () => operatorApi.addVerdict(review.id, topic, words, true), open && Boolean(words));
  return (
    <li className="rd-row">
      <Link to={`/topics/${encodeURIComponent(topic)}`} className="rd-row-link">
        <span className="font-semibold">{topic}</span>
      </Link>
      {recorded.map((v, n) => {
        const at = toDate(v.at);
        return (
          <div key={n} className="rd-answer">
            {at ? <span className="text-caption text-app-muted-foreground"><When date={at} /></span> : null}
            <Markdown content={v.body} />
          </div>
        );
      })}
      <EntryField label={t(strings.reviews.verdictLabel)} rows={3} value={text} onChange={setText} />
      <div>
        <Button variant="secondary" disabled={!text.trim() || entryTooLong(text)} onClick={() => setOpen(true)}>{t(strings.reviews.record)}</Button>
      </div>
      <Confirm
        open={open}
        title={t(strings.reviews.confirmTitle, { topic })}
        description={t(strings.reviews.confirmDescription)}
        confirmLabel={t(strings.reviews.record)}
        onClose={() => setOpen(false)}
        onConfirm={async () => {
          await operatorApi.addVerdict(review.id, topic, text.trim());
          setText("");
          await client.invalidateQueries({ queryKey: ["operator", "reviews"] });
        }}
      >
        <WouldWrite label={t(strings.reviews.appendsTo, { path: `${review.folder}/VERDICTS.md` })} query={preview} />
      </Confirm>
    </li>
  );
}

/** A verdict on a topic the review covers without a section of its own. */
function OtherTopic({ exclude, onPick }: { exclude: string[]; onPick: (topic: string) => void }) {
  const { t } = useTranslation();
  const topics = useTopics().data?.topics.filter((x) => !exclude.includes(x.topic)) ?? [];
  if (topics.length === 0) return null;
  return (
    <label className="flex max-w-md flex-col gap-space-2xs text-body-sm">
      <span className="font-medium">{t(strings.reviews.otherTopic)}</span>
      <span className="text-app-muted-foreground">{t(strings.reviews.otherTopicNote)}</span>
      <Select
        value=""
        onValueChange={(value) => value && onPick(value)}
        options={[{ value: "", label: t(strings.reviews.pickTopic) }, ...topics.map((x) => ({ value: x.topic, label: x.title?.value || x.topic }))]}
      />
    </label>
  );
}
