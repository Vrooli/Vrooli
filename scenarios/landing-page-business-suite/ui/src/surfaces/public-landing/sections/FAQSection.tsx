import { useState } from 'react';
import { ChevronDown } from 'lucide-react';

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
}

export function FAQSection({ content }: FAQSectionProps) {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  const faqs = content.faqs || [
    {
      question: 'What is Silent Founder OS?',
      answer: 'It is the Vrooli business bundle built for solo operators. The first app is Vrooli Ascension: it automates browser workflows and exports studio-quality screen recordings for marketing.',
    },
    {
      question: 'Do I need to code or design?',
      answer: 'No. BAS is visual and AI-assisted. You point it at your browser tasks, it builds the workflow, retries failures, and records polished walkthroughs with overlays.',
    },
    {
      question: 'What can BAS automate?',
      answer: 'Lead research, CRM updates, onboarding checklists, ad dashboard pulls, and product walkthroughs. Anything in a browser with clicks, waits, loops, and extracts.',
    },
    {
      question: 'How are screen recordings produced?',
      answer: 'Every automation run can output HD recordings with branded overlays. When your UI changes, re-run the flow and get fresh reels automatically.',
    },
    {
      question: 'What happens as new apps launch?',
      answer: 'Early adopters keep bundle pricing and gain access as we ship new business apps. Your automations, assets, and analytics stay in the same OS.',
    },
  ];

  return (
    <section className="bg-[#07090F] py-24">
      <div className="container mx-auto px-6">
        <div className="max-w-3xl">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">FAQ</p>
          <h2 className="mt-3 text-4xl font-semibold text-white">
            {content.title || 'Guardrails teams ask about before launch'}
          </h2>
          <p className="mt-3 text-lg text-slate-300">
            {content.subtitle || 'From staging folders to download gating, here are the answers ops leaders reference most often.'}
          </p>
        </div>

        <div className="mt-10 space-y-4">
          {faqs.map((faq, index) => (
            <div
              key={index}
              className="rounded-2xl border border-white/10 bg-[#0F172A] transition-all duration-200"
              data-testid={`faq-${index}`}
            >
              <button
                onClick={() => setOpenIndex(openIndex === index ? null : index)}
                className="flex w-full items-center justify-between px-6 py-5 text-left"
                data-testid={`faq-question-${index}`}
              >
                <span className="pr-8 text-lg font-semibold text-white">{faq.question}</span>
                <ChevronDown
                  className={`h-5 w-5 text-[#F97316] transition-transform ${openIndex === index ? 'rotate-180' : ''}`}
                />
              </button>
              <div
                className={`overflow-hidden transition-all duration-300 ${
                  openIndex === index ? 'max-h-96' : 'max-h-0'
                }`}
                data-testid={`faq-answer-${index}`}
              >
                <div className="px-6 pb-6 text-sm text-slate-300">{faq.answer}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
