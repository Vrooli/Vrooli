import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { MessageSquare, RefreshCcw, Bug, Lightbulb, Heart, Send, CheckCircle, ArrowLeft } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';

type FeedbackType = 'refund' | 'bug' | 'feature' | 'general';

interface FeedbackFormData {
  type: FeedbackType;
  email: string;
  subject: string;
  message: string;
  orderId?: string;
}

const feedbackTypes: { value: FeedbackType; label: string; icon: React.ReactNode; description: string }[] = [
  {
    value: 'refund',
    label: 'Request a Refund',
    icon: <RefreshCcw className="h-5 w-5" />,
    description: '30-day money-back guarantee, no questions asked',
  },
  {
    value: 'bug',
    label: 'Report a Bug',
    icon: <Bug className="h-5 w-5" />,
    description: 'Help us fix issues you encounter',
  },
  {
    value: 'feature',
    label: 'Feature Request',
    icon: <Lightbulb className="h-5 w-5" />,
    description: 'Share ideas for new capabilities',
  },
  {
    value: 'general',
    label: 'General Feedback',
    icon: <Heart className="h-5 w-5" />,
    description: 'Tell us what you think',
  },
];

export function FeedbackPage() {
  const navigate = useNavigate();
  const [form, setForm] = useState<FeedbackFormData>({
    type: 'general',
    email: '',
    subject: '',
    message: '',
    orderId: '',
  });
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      const response = await fetch('/api/feedback', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });

      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to submit feedback');
      }

      setSubmitted(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const selectedType = feedbackTypes.find((t) => t.value === form.type);

  if (submitted) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-indigo-950 text-white">
        <div className="mx-auto max-w-2xl px-6 py-16">
          <Card className="bg-slate-900 border-slate-800 shadow-2xl shadow-indigo-900/30">
            <CardContent className="flex flex-col items-center gap-6 py-12">
              <div className="rounded-full bg-emerald-500/20 p-4">
                <CheckCircle className="h-12 w-12 text-emerald-400" />
              </div>
              <div className="text-center">
                <h2 className="text-2xl font-semibold">Thank you for your feedback!</h2>
                <p className="mt-2 text-slate-300">
                  {form.type === 'refund'
                    ? "We've received your refund request. You'll hear from us within 1-2 business days."
                    : "We've received your message and will get back to you if needed."}
                </p>
              </div>
              <Button onClick={() => navigate('/')} variant="outline" className="border-white/20 text-white hover:border-white/40">
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to home
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-indigo-950 text-white">
      <div className="mx-auto max-w-3xl px-6 py-16 space-y-8">
        <div className="flex flex-col gap-3">
          <button
            type="button"
            onClick={() => navigate('/')}
            className="self-start rounded-full border border-white/10 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-300 hover:border-white/30"
          >
            <ArrowLeft className="mr-1 inline h-3 w-3" />
            Back to home
          </button>
          <h1 className="text-4xl font-semibold leading-tight">Feedback & Support</h1>
          <p className="text-lg text-slate-300">
            We're always looking to improve. Let us know how we can help.
          </p>
        </div>

        <Card className="bg-slate-900 border-slate-800 shadow-2xl shadow-indigo-900/30">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MessageSquare className="h-5 w-5 text-orange-400" />
              Send us a message
            </CardTitle>
            <CardDescription className="text-slate-300">
              Select a category and tell us what's on your mind.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Feedback Type Selection */}
              <div className="grid gap-3 sm:grid-cols-2">
                {feedbackTypes.map((type) => (
                  <button
                    key={type.value}
                    type="button"
                    onClick={() => setForm((f) => ({ ...f, type: type.value }))}
                    className={`flex items-start gap-3 rounded-xl border p-4 text-left transition-all ${
                      form.type === type.value
                        ? 'border-orange-500 bg-orange-500/10'
                        : 'border-white/10 bg-white/5 hover:border-white/20'
                    }`}
                  >
                    <div className={`rounded-lg p-2 ${form.type === type.value ? 'bg-orange-500/20 text-orange-400' : 'bg-white/10 text-slate-400'}`}>
                      {type.icon}
                    </div>
                    <div>
                      <p className="font-medium text-white">{type.label}</p>
                      <p className="text-xs text-slate-400">{type.description}</p>
                    </div>
                  </button>
                ))}
              </div>

              {/* Email */}
              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Email address <span className="text-rose-400">*</span>
                </label>
                <input
                  type="email"
                  required
                  value={form.email}
                  onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                  placeholder="you@example.com"
                  className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-3 text-white placeholder-slate-500 focus:border-orange-500 focus:outline-none focus:ring-1 focus:ring-orange-500"
                />
              </div>

              {/* Order ID for refunds */}
              {form.type === 'refund' && (
                <div>
                  <label className="block text-sm font-medium text-slate-300">
                    Order or Subscription ID <span className="text-slate-500">(optional)</span>
                  </label>
                  <input
                    type="text"
                    value={form.orderId}
                    onChange={(e) => setForm((f) => ({ ...f, orderId: e.target.value }))}
                    placeholder="sub_xxxxx or cs_xxxxx"
                    className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-3 text-white placeholder-slate-500 focus:border-orange-500 focus:outline-none focus:ring-1 focus:ring-orange-500"
                  />
                  <p className="mt-1 text-xs text-slate-500">
                    You can find this in your Stripe receipt email or billing portal.
                  </p>
                </div>
              )}

              {/* Subject */}
              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Subject <span className="text-rose-400">*</span>
                </label>
                <input
                  type="text"
                  required
                  value={form.subject}
                  onChange={(e) => setForm((f) => ({ ...f, subject: e.target.value }))}
                  placeholder={
                    form.type === 'refund'
                      ? 'Refund request for my subscription'
                      : form.type === 'bug'
                        ? 'Bug: Describe the issue briefly'
                        : form.type === 'feature'
                          ? 'Feature idea: Your suggestion'
                          : 'How can we help?'
                  }
                  className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-3 text-white placeholder-slate-500 focus:border-orange-500 focus:outline-none focus:ring-1 focus:ring-orange-500"
                />
              </div>

              {/* Message */}
              <div>
                <label className="block text-sm font-medium text-slate-300">
                  Message <span className="text-rose-400">*</span>
                </label>
                <textarea
                  required
                  rows={5}
                  value={form.message}
                  onChange={(e) => setForm((f) => ({ ...f, message: e.target.value }))}
                  placeholder={
                    form.type === 'refund'
                      ? "Tell us why you'd like a refund (optional but helpful for us to improve)."
                      : form.type === 'bug'
                        ? 'Describe the bug: What happened? What did you expect to happen? Steps to reproduce?'
                        : form.type === 'feature'
                          ? 'Describe your feature idea and how it would help you.'
                          : 'Share your thoughts, questions, or suggestions.'
                  }
                  className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-3 text-white placeholder-slate-500 focus:border-orange-500 focus:outline-none focus:ring-1 focus:ring-orange-500 resize-none"
                />
              </div>

              {error && (
                <div className="rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 text-sm text-rose-300">
                  {error}
                </div>
              )}

              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <p className="text-xs text-slate-500">
                  {form.type === 'refund'
                    ? 'Refunds are processed within 5-7 business days after approval.'
                    : 'We typically respond within 1-2 business days.'}
                </p>
                <Button
                  type="submit"
                  disabled={submitting}
                  className="bg-orange-500 hover:bg-orange-600 text-white"
                >
                  {submitting ? (
                    'Sending...'
                  ) : (
                    <>
                      <Send className="mr-2 h-4 w-4" />
                      Send {selectedType?.label.split(' ')[0] || 'Message'}
                    </>
                  )}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        {/* Additional info for refunds */}
        {form.type === 'refund' && (
          <Card className="border-emerald-500/30 bg-emerald-500/5">
            <CardContent className="py-4">
              <div className="flex items-start gap-3">
                <CheckCircle className="mt-0.5 h-5 w-5 flex-shrink-0 text-emerald-400" />
                <div>
                  <p className="font-medium text-emerald-300">30-Day Money-Back Guarantee</p>
                  <p className="mt-1 text-sm text-emerald-200/80">
                    If you're not satisfied within 30 days of your purchase, we'll refund your payment—no questions asked.
                    Just submit this form with your email address.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
