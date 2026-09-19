import { useId, useRef, useState, type FormEvent, type ReactNode } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Code, ConnectError } from '@connectrpc/connect';
import { AlertCircle, Bug, Clock, Lightbulb, Mail, MapPin, MessageCircle, RefreshCcw, Send } from 'lucide-react';
import { createFeedback } from '../../../shared/api/feedback';
import { SiteShell } from './SiteShell';
import { useSiteIdentity } from './useSiteIdentity';
import { siteCopy } from './siteCopy';
import { SUBMIT_TIMEOUT_MS, validateContact, type ContactField, type ContactFieldErrors } from './contactValidation';

const copy = siteCopy.contactPage;
type Topic = keyof typeof siteCopy.topics;
type Field = ContactField;
type FieldErrors = ContactFieldErrors;

const TOPICS: { value: Topic; icon: ReactNode }[] = [
  { value: 'general', icon: <MessageCircle aria-hidden="true" /> },
  { value: 'bug', icon: <Bug aria-hidden="true" /> },
  { value: 'feature', icon: <Lightbulb aria-hidden="true" /> },
  { value: 'refund', icon: <RefreshCcw aria-hidden="true" /> },
];

export function ContactPage() {
  const identity = useSiteIdentity();
  const navigate = useNavigate();
  const id = useId();
  const [params] = useSearchParams();
  // /contact?topic=refund deep-links straight into a refund request.
  const [topic, setTopic] = useState<Topic>(() => { const requested = params.get('topic'); return requested && requested in siteCopy.topics ? requested as Topic : 'general'; });
  const [values, setValues] = useState({ email: '', subject: '', message: '', orderId: '' });
  const [errors, setErrors] = useState<FieldErrors>({});
  const [touched, setTouched] = useState<Partial<Record<Field, boolean>>>({});
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const summary = useRef<HTMLDivElement>(null);
  const fields = { email: useRef<HTMLInputElement>(null), subject: useRef<HTMLInputElement>(null), message: useRef<HTMLTextAreaElement>(null) };

  const update = (field: keyof typeof values, value: string) => {
    const next = { ...values, [field]: value };
    setValues(next);
    setSubmitError(null);
    // Re-validate a field the visitor has already been told about, so the
    // message clears as soon as the input is fixed rather than on next submit.
    if (field !== 'orderId' && (touched[field] || errors[field])) setErrors(current => ({ ...current, [field]: validateContact(next)[field] }));
  };
  const blur = (field: Field) => {
    if (!values[field].trim() && !errors[field]) return; // do not scold an untouched empty field
    setTouched(current => ({ ...current, [field]: true }));
    setErrors(current => ({ ...current, [field]: validateContact(values)[field] }));
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (submitting) return;
    const found = validateContact(values);
    setErrors(found);
    setTouched({ email: true, subject: true, message: true });
    const firstInvalid = (['email', 'subject', 'message'] as Field[]).find(field => found[field]);
    if (firstInvalid) { fields[firstInvalid].current?.focus(); return; }
    setSubmitting(true);
    setSubmitError(null);
    const controller = new AbortController();
    const timer = setTimeout(() => { controller.abort(); }, SUBMIT_TIMEOUT_MS);
    try {
      await createFeedback({ type: topic, email: values.email.trim(), subject: values.subject.trim(), message: values.message.trim(), ...(topic === 'refund' && values.orderId.trim() ? { orderId: values.orderId.trim() } : {}) }, controller.signal);
      navigate(`/thank-you?type=${topic === 'refund' ? 'refund' : 'message'}`);
    } catch (error) {
      const connectError = error instanceof ConnectError ? error : undefined;
      if (controller.signal.aborted || (error instanceof Error && error.name === 'AbortError')) setSubmitError(copy.errors.timeout);
      else if (error instanceof TypeError || connectError?.code === Code.Unavailable) setSubmitError(copy.errors.network);
      else if (connectError?.code === Code.InvalidArgument && connectError.rawMessage) setSubmitError(connectError.rawMessage.charAt(0).toUpperCase() + connectError.rawMessage.slice(1) + '.');
      else setSubmitError(copy.errors.server);
      setTimeout(() => { summary.current?.focus(); }, 0);
    } finally {
      clearTimeout(timer);
      setSubmitting(false);
    }
  };

  const described = (field: Field, hint?: boolean) => [hint ? `${id}-${field}-hint` : '', errors[field] ? `${id}-${field}-error` : ''].filter(Boolean).join(' ') || undefined;
  const errorFor = (field: Field) => errors[field] && <p className="site-field-error" id={`${id}-${field}-error`}><AlertCircle aria-hidden="true" />{errors[field]}</p>;

  return <SiteShell meta={{ title: copy.title, description: copy.description }} identity={identity}>
    <div className="site-contact">
      <section className="site-hero site-contact-intro" aria-labelledby={`${id}-heading`}>
        <p className="eyebrow">{copy.eyebrow}</p>
        <h1 id={`${id}-heading`}>{copy.heading}</h1>
        <p className="site-lede">{copy.lede}</p>
        <ul className="site-contact-facts">
          <li><Clock aria-hidden="true" /><span>{copy.responseTime}</span></li>
          {identity.contactEmail && <li><Mail aria-hidden="true" /><span><strong>{copy.emailHeading}</strong><a href={`mailto:${identity.contactEmail}`}>{identity.contactEmail}</a></span></li>}
          {identity.addressLines.length > 0 && <li><MapPin aria-hidden="true" /><span><strong>{copy.addressHeading}</strong><address>{identity.addressLines.map((line, index) => <span key={index}>{line}</span>)}</address></span></li>}
          <li><RefreshCcw aria-hidden="true" /><span><strong>{copy.refundHeading}</strong>{copy.refundBody}</span></li>
        </ul>
      </section>

      <form className="site-card site-form" noValidate onSubmit={event => { void submit(event); }} aria-labelledby={`${id}-form-heading`} aria-busy={submitting}>
        <h2 id={`${id}-form-heading`}>{copy.formHeading}</h2>
        <div ref={summary} tabIndex={-1} className="site-form-alert-slot" aria-live="assertive">
          {submitError && <div className="site-alert" role="alert"><AlertCircle aria-hidden="true" /><p>{submitError}</p></div>}
        </div>

        <fieldset className="site-topics" disabled={submitting}>
          <legend>{copy.topicLegend}</legend>
          <div className="site-topic-grid">
            {TOPICS.map(({ value, icon }) => <label key={value} className="site-topic" data-selected={topic === value}>
              <input type="radio" name="topic" value={value} checked={topic === value} onChange={() => { setTopic(value); setSubmitError(null); }} />
              <span className="site-topic-icon">{icon}</span>
              <span><strong>{siteCopy.topics[value].label}</strong><small>{siteCopy.topics[value].description}</small></span>
            </label>)}
          </div>
        </fieldset>

        <div className="site-field" data-invalid={Boolean(errors.email)}>
          <label htmlFor={`${id}-email`}>{copy.email}</label>
          <input ref={fields.email} id={`${id}-email`} type="email" name="email" autoComplete="email" inputMode="email" required disabled={submitting}
            value={values.email} onChange={event => { update('email', event.target.value); }} onBlur={() => { blur('email'); }}
            aria-invalid={Boolean(errors.email)} aria-describedby={described('email', true)} placeholder="name@example.com" />
          <p className="site-field-hint" id={`${id}-email-hint`}>{copy.emailHint}</p>
          {errorFor('email')}
        </div>

        {topic === 'refund' && <div className="site-field">
          <label htmlFor={`${id}-order`}>{copy.orderId} <span className="site-optional">({copy.optional})</span></label>
          <input id={`${id}-order`} type="text" name="orderId" disabled={submitting} value={values.orderId} onChange={event => { update('orderId', event.target.value); }}
            aria-describedby={`${id}-order-hint`} placeholder="sub_… or cs_…" />
          <p className="site-field-hint" id={`${id}-order-hint`}>{copy.orderHint}</p>
        </div>}

        <div className="site-field" data-invalid={Boolean(errors.subject)}>
          <label htmlFor={`${id}-subject`}>{copy.subject}</label>
          <input ref={fields.subject} id={`${id}-subject`} type="text" name="subject" required maxLength={240} disabled={submitting}
            value={values.subject} onChange={event => { update('subject', event.target.value); }} onBlur={() => { blur('subject'); }}
            aria-invalid={Boolean(errors.subject)} aria-describedby={described('subject')} />
          {errorFor('subject')}
        </div>

        <div className="site-field" data-invalid={Boolean(errors.message)}>
          <label htmlFor={`${id}-message`}>{copy.message}</label>
          <textarea ref={fields.message} id={`${id}-message`} name="message" rows={6} required disabled={submitting}
            value={values.message} onChange={event => { update('message', event.target.value); }} onBlur={() => { blur('message'); }}
            aria-invalid={Boolean(errors.message)} aria-describedby={described('message')} />
          {errorFor('message')}
        </div>

        <div className="site-form-actions">
          <button type="submit" className="button button-primary" disabled={submitting} aria-disabled={submitting}>
            {submitting ? <><span className="site-spinner" aria-hidden="true" />{copy.submitting}</> : <>{copy.submit}<Send aria-hidden="true" /></>}
          </button>
          <span className="sr-only" role="status">{submitting ? copy.submitting : ''}</span>
        </div>
      </form>
    </div>
  </SiteShell>;
}
