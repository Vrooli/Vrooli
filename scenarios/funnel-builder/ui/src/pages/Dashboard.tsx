import { useNavigate } from 'react-router-dom'
import { Plus, TrendingUp, Users, Target, Clock } from 'lucide-react'
import { useEffect } from 'react'
import { useFunnelStore } from '../store/useFunnelStore'

const Dashboard = () => {
  const navigate = useNavigate()
  const { funnels, setFunnels } = useFunnelStore()

  useEffect(() => {
    // Mock data for demonstration
    setFunnels([
      {
        id: '1',
        name: 'Lead Generation Funnel',
        slug: 'lead-gen',
        description: 'Capture high-quality leads',
        steps: [],
        settings: { theme: { primaryColor: '#0ea5e9' }, progressBar: true },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        status: 'active'
      },
      {
        id: '2',
        name: 'Product Launch Funnel',
        slug: 'product-launch',
        description: 'Drive sales for new product',
        steps: [],
        settings: { theme: { primaryColor: '#0ea5e9' }, progressBar: true },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        status: 'draft'
      }
    ])
  }, [])

  const stats = [
    { label: 'Total Funnels', value: funnels.length, icon: Target, color: 'text-blue-600' },
    { label: 'Total Leads', value: '1,234', icon: Users, color: 'text-green-600' },
    { label: 'Avg. Conversion', value: '23.4%', icon: TrendingUp, color: 'text-purple-600' },
    { label: 'Avg. Time', value: '2m 45s', icon: Clock, color: 'text-orange-600' },
  ]

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6 sm:mb-8">
        <h2 className="text-2xl font-bold text-gray-900 sm:text-3xl mb-2">Dashboard</h2>
        <p className="text-gray-600 text-sm sm:text-base">Manage your funnels and track performance</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:gap-6 md:grid-cols-2 lg:grid-cols-4 mb-6 sm:mb-8">
        {stats.map((stat) => (
          <div key={stat.label} className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <stat.icon className={`w-8 h-8 ${stat.color}`} />
            </div>
            <p className="text-2xl font-bold text-gray-900 mb-1">{stat.value}</p>
            <p className="text-sm text-gray-600">{stat.label}</p>
          </div>
        ))}
      </div>

      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h3 className="text-lg font-bold text-gray-900 sm:text-xl">Your Funnels</h3>
        <button
          onClick={() => navigate('/builder')}
          className="btn btn-primary"
        >
          <Plus className="w-4 h-4 mr-2" />
          Create Funnel
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:gap-6 md:grid-cols-2 lg:grid-cols-3">
        {funnels.map((funnel) => (
          <div key={funnel.id} className="card p-6 hover:shadow-lg transition-shadow cursor-pointer"
               onClick={() => navigate(`/builder/${funnel.id}`)}>
            <div className="flex justify-between items-start mb-4">
              <h4 className="text-lg font-semibold text-gray-900">{funnel.name}</h4>
              <span className={`px-2 py-1 text-xs rounded-full ${
                funnel.status === 'active' ? 'bg-green-100 text-green-700' :
                funnel.status === 'draft' ? 'bg-yellow-100 text-yellow-700' :
                'bg-gray-100 text-gray-700'
              }`}>
                {funnel.status}
              </span>
            </div>
            <p className="text-gray-600 text-sm mb-4">{funnel.description}</p>
            <div className="flex justify-between text-sm text-gray-500">
              <span>{funnel.steps.length} steps</span>
              <span>Updated {new Date(funnel.updatedAt).toLocaleDateString()}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default Dashboard
