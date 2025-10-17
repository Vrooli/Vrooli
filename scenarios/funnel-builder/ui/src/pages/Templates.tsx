import { useNavigate } from 'react-router-dom'
import { FileText, Users, Zap, TrendingUp, Gift, Calendar, ShoppingCart, Rocket } from 'lucide-react'

const Templates = () => {
  const navigate = useNavigate()

  const templates = [
    {
      id: 'lead-gen',
      name: 'Lead Generation',
      description: 'Capture high-quality leads with qualifying questions',
      icon: Users,
      color: 'bg-blue-500',
      steps: 5,
      conversionRate: '32%',
      category: 'Marketing'
    },
    {
      id: 'product-launch',
      name: 'Product Launch',
      description: 'Build excitement and drive sales for new products',
      icon: Rocket,
      color: 'bg-purple-500',
      steps: 7,
      conversionRate: '28%',
      category: 'Sales'
    },
    {
      id: 'webinar-reg',
      name: 'Webinar Registration',
      description: 'Get attendees for your online events and webinars',
      icon: Calendar,
      color: 'bg-green-500',
      steps: 4,
      conversionRate: '45%',
      category: 'Events'
    },
    {
      id: 'quiz-funnel',
      name: 'Interactive Quiz',
      description: 'Engage visitors with personalized quiz results',
      icon: Zap,
      color: 'bg-yellow-500',
      steps: 6,
      conversionRate: '38%',
      category: 'Engagement'
    },
    {
      id: 'free-trial',
      name: 'Free Trial Signup',
      description: 'Convert visitors into trial users for SaaS products',
      icon: Gift,
      color: 'bg-pink-500',
      steps: 5,
      conversionRate: '25%',
      category: 'SaaS'
    },
    {
      id: 'ecommerce',
      name: 'E-commerce Checkout',
      description: 'Optimize your checkout process for higher sales',
      icon: ShoppingCart,
      color: 'bg-indigo-500',
      steps: 4,
      conversionRate: '42%',
      category: 'E-commerce'
    },
    {
      id: 'survey',
      name: 'Customer Survey',
      description: 'Collect valuable feedback from your customers',
      icon: FileText,
      color: 'bg-red-500',
      steps: 5,
      conversionRate: '55%',
      category: 'Feedback'
    },
    {
      id: 'roi-calculator',
      name: 'ROI Calculator',
      description: 'Help prospects calculate their potential returns',
      icon: TrendingUp,
      color: 'bg-orange-500',
      steps: 6,
      conversionRate: '35%',
      category: 'Tools'
    }
  ]

  const categories = ['All', 'Marketing', 'Sales', 'Events', 'Engagement', 'SaaS', 'E-commerce', 'Feedback', 'Tools']

  const handleUseTemplate = (templateId: string) => {
    // Create new funnel from template
    navigate(`/builder/new?template=${templateId}`)
  }

  return (
    <div className="p-8">
      <div className="mb-8">
        <h2 className="text-3xl font-bold text-gray-900 mb-2">Funnel Templates</h2>
        <p className="text-gray-600">Start with a proven template and customize it to your needs</p>
      </div>

      <div className="mb-6">
        <div className="flex gap-2 flex-wrap">
          {categories.map((category) => (
            <button
              key={category}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                category === 'All'
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {category}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {templates.map((template) => (
          <div
            key={template.id}
            className="card hover:shadow-lg transition-shadow cursor-pointer group"
            onClick={() => handleUseTemplate(template.id)}
          >
            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <div className={`p-3 rounded-lg ${template.color}`}>
                  <template.icon className="w-6 h-6 text-white" />
                </div>
                <span className="px-2 py-1 bg-gray-100 text-gray-600 text-xs rounded-full">
                  {template.category}
                </span>
              </div>
              
              <h3 className="text-lg font-semibold text-gray-900 mb-2 group-hover:text-primary-600 transition-colors">
                {template.name}
              </h3>
              
              <p className="text-sm text-gray-600 mb-4">
                {template.description}
              </p>
              
              <div className="flex justify-between items-center pt-4 border-t border-gray-100">
                <div className="flex gap-4 text-sm">
                  <span className="text-gray-500">
                    <span className="font-medium text-gray-700">{template.steps}</span> steps
                  </span>
                  <span className="text-gray-500">
                    <span className="font-medium text-green-600">{template.conversionRate}</span> conv.
                  </span>
                </div>
              </div>
            </div>
            
            <div className="px-6 pb-4">
              <button className="btn btn-primary w-full opacity-0 group-hover:opacity-100 transition-opacity">
                Use Template
              </button>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-12 p-8 bg-gradient-to-r from-primary-50 to-blue-50 rounded-2xl">
        <div className="max-w-3xl">
          <h3 className="text-2xl font-bold text-gray-900 mb-3">
            Can't find what you're looking for?
          </h3>
          <p className="text-gray-600 mb-6">
            Start from scratch and build your own custom funnel with our intuitive drag-and-drop builder.
          </p>
          <button
            onClick={() => navigate('/builder')}
            className="btn btn-primary"
          >
            Create Custom Funnel
          </button>
        </div>
      </div>
    </div>
  )
}

export default Templates