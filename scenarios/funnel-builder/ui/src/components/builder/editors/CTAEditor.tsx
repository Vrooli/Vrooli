import { CTAContent } from '../../../types'
import { AlertCircle } from 'lucide-react'

interface CTAEditorProps {
  content: CTAContent
  onChange: (content: CTAContent) => void
}

const CTAEditor = ({ content, onChange }: CTAEditorProps) => {
  return (
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Call to Action</h3>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Headline
          </label>
          <input
            type="text"
            value={content.headline || ''}
            onChange={(e) => onChange({ ...content, headline: e.target.value })}
            className="input w-full"
            placeholder="Your compelling offer headline"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Subheadline (Optional)
          </label>
          <input
            type="text"
            value={content.subheadline || ''}
            onChange={(e) => onChange({ ...content, subheadline: e.target.value })}
            className="input w-full"
            placeholder="Supporting text for your offer"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Button Text
            </label>
            <input
              type="text"
              value={content.buttonText || ''}
              onChange={(e) => onChange({ ...content, buttonText: e.target.value })}
              className="input w-full"
              placeholder="Get Started Now"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Button URL
            </label>
            <input
              type="url"
              value={content.buttonUrl || ''}
              onChange={(e) => onChange({ ...content, buttonUrl: e.target.value })}
              className="input w-full"
              placeholder="https://example.com/checkout"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Urgency Message (Optional)
          </label>
          <div className="flex gap-2">
            <AlertCircle className="w-5 h-5 text-orange-500 flex-shrink-0 mt-2" />
            <input
              type="text"
              value={content.urgency || ''}
              onChange={(e) => onChange({ ...content, urgency: e.target.value })}
              className="input flex-1"
              placeholder="Limited time offer - expires in 24 hours!"
            />
          </div>
          <p className="text-xs text-gray-500 mt-1 ml-7">
            Add urgency to increase conversions (use sparingly)
          </p>
        </div>

        <div className="p-4 bg-blue-50 rounded-lg border border-blue-200">
          <h4 className="text-sm font-medium text-blue-900 mb-2">Preview</h4>
          <div className="bg-white rounded-lg p-6 text-center border border-blue-200">
            <h2 className="text-2xl font-bold text-gray-900 mb-2">
              {content.headline || 'Your Headline Here'}
            </h2>
            {content.subheadline && (
              <p className="text-gray-600 mb-4">{content.subheadline}</p>
            )}
            {content.urgency && (
              <div className="inline-flex items-center gap-2 px-3 py-1 bg-orange-100 text-orange-700 rounded-full text-sm mb-4">
                <AlertCircle className="w-4 h-4" />
                {content.urgency}
              </div>
            )}
            <div>
              <button className="btn btn-primary px-8 py-3 text-lg">
                {content.buttonText || 'Click Here'}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default CTAEditor