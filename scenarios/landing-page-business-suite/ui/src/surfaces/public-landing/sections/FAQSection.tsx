import { useState } from 'react';
import { ChevronDown, Bot } from 'lucide-react';
import { Button } from '../../../shared/ui/button';

interface FAQItem {
  question: string;
  answer: string;
}

interface FAQSectionProps {
  content: {
    title?: string;
    subtitle?: string;
    faqs?: FAQItem[];
  };
  supportChatUrl?: string | null;
}

export function FAQSection({ content, supportChatUrl }: FAQSectionProps) {
  const [openIndices, setOpenIndices] = useState<Set<number>>(new Set([0]));

  const toggleFaq = (index: number) => {
    setOpenIndices((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  };

  const faqs = content.faqs || [
    {
      question: 'What is Vrooli Ascension?',
      answer: 'Vrooli Ascension automates browser workflows and exports studio-quality screen recordings. Record your browser actions once, then replay them forever—no coding required. Perfect for product demos, onboarding walkthroughs, and repetitive browser tasks.',
    },
    {
      question: 'Do I need technical skills to use it?',
      answer: 'Not at all. Vrooli Ascension records your browser actions as you work—just click, type, and navigate normally. The system captures everything and lets you turn any sequence into a reusable automation or polished video.',
    },
    {
      question: 'What kinds of tasks can I automate?',
      answer: 'Anything you do in a browser: lead research, CRM updates, onboarding checklists, dashboard reporting, product walkthroughs, and more. If it involves clicks, typing, and navigation, Vrooli can automate it.',
    },
    {
      question: 'How does pricing work?',
      answer: 'Simple flat pricing with no per-seat fees. Pick a plan that fits your usage, and you get access to all features at that tier. As we ship new apps in the Vrooli suite, your subscription includes them automatically at no extra cost.',
    },
    {
      question: 'Can I cancel anytime?',
      answer: "Yes, cancel anytime with no questions asked. Your subscription stays active until the end of your billing period, and you keep access to everything until then. We also offer a 30-day money-back guarantee if you're not satisfied.",
    },
    {
      question: 'Is my data secure?',
      answer: 'Absolutely. Vrooli runs locally on your machine—your browser data and recordings never leave your computer unless you explicitly export or share them. No cloud dependency means you stay in full control.',
    },
    {
      question: 'What support is available?',
      answer: 'All plans include async support via email. Pro plans and above get priority responses. We also maintain detailed documentation and video tutorials to help you get started quickly.',
    },
    {
      question: 'What happens when new apps launch?',
      answer: 'Subscribers automatically get access to new Vrooli suite apps as we release them—at no additional cost. Your workflows, assets, and data stay connected across all apps in the ecosystem.',
    },
  ];

  // Split FAQs into two columns for desktop
  const midpoint = Math.ceil(faqs.length / 2);
  const leftColumnFaqs = faqs.slice(0, midpoint);
  const rightColumnFaqs = faqs.slice(midpoint);

  const renderFaqItem = (faq: FAQItem, index: number) => (
    <div
      key={index}
      className="rounded-2xl border border-white/10 bg-[#0F172A] transition-all duration-200 hover:border-white/20"
      data-testid={`faq-${index}`}
    >
      <button
        onClick={() => toggleFaq(index)}
        className="flex w-full items-center justify-between px-6 py-5 text-left"
        data-testid={`faq-question-${index}`}
      >
        <span className="pr-4 text-base font-semibold text-white">{faq.question}</span>
        <ChevronDown
          className={`h-5 w-5 flex-shrink-0 text-[#F97316] transition-transform duration-200 ${openIndices.has(index) ? 'rotate-180' : ''}`}
        />
      </button>
      <div
        className={`overflow-hidden transition-all duration-300 ${
          openIndices.has(index) ? 'max-h-96' : 'max-h-0'
        }`}
        data-testid={`faq-answer-${index}`}
      >
        <div className="px-6 pb-6 text-sm leading-relaxed text-slate-300">{faq.answer}</div>
      </div>
    </div>
  );

  return (
    <section className="bg-[#07090F] py-24">
      <div className="container mx-auto px-6">
        <div className="max-w-3xl">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">FAQ</p>
          <h2 className="mt-3 text-4xl font-semibold text-white">
            {content.title || 'Frequently asked questions'}
          </h2>
          <p className="mt-3 text-lg text-slate-300">
            {content.subtitle || 'Everything you need to know before getting started.'}
          </p>
        </div>

        {/* Two-column layout on desktop, single column on mobile */}
        <div className="mt-10 grid gap-4 lg:grid-cols-2 lg:gap-6">
          <div className="space-y-4">
            {leftColumnFaqs.map((faq, index) => renderFaqItem(faq, index))}
          </div>
          <div className="space-y-4">
            {rightColumnFaqs.map((faq, index) => renderFaqItem(faq, index + midpoint))}
          </div>
        </div>

        {/* AI Assistant CTA - only show if supportChatUrl is configured */}
        {supportChatUrl && (
          <div className="mt-12 flex flex-col items-center justify-center gap-4 rounded-2xl border border-white/10 bg-[#0F172A] p-8 text-center">
            <Bot className="h-8 w-8 text-[#F97316]" />
            <div>
              <h3 className="text-lg font-semibold text-white">Still have questions?</h3>
              <p className="mt-1 text-sm text-slate-300">Get instant answers from our AI assistant—no waiting, no calls.</p>
            </div>
            <Button asChild variant="outline" size="sm" className="border-white/20 text-white hover:border-white/40">
              <a href={supportChatUrl} target="_blank" rel="noopener noreferrer">Ask our AI assistant</a>
            </Button>
          </div>
        )}
      </div>
    </section>
  );
}
