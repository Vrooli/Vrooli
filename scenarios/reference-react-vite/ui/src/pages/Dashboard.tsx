import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ListTodo, FolderKanban, CheckCircle2, Clock, AlertCircle } from "lucide-react";
import { fetchTasks, fetchProjects } from "../lib/api";
import { Button } from "../components/ui/button";
import { ErrorBoundary } from "../components/ErrorBoundary";

function StatCard({
  icon: Icon,
  label,
  value,
  loading
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: number | string;
  loading?: boolean;
}) {
  return (
    <div
      data-testid={`stat-${label.toLowerCase().replace(/\s+/g, "-")}`}
      className="rounded-xl border border-white/10 bg-white/5 p-4"
    >
      <div className="flex items-center gap-3">
        <div className="rounded-lg bg-white/10 p-2">
          <Icon className="h-5 w-5 text-slate-300" />
        </div>
        <div>
          <p className="text-sm text-slate-400">{label}</p>
          {loading ? (
            <p className="text-2xl font-semibold text-slate-500">--</p>
          ) : (
            <p className="text-2xl font-semibold">{value}</p>
          )}
        </div>
      </div>
    </div>
  );
}

function RecentTasks() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["tasks", { limit: 5 }],
    queryFn: () => fetchTasks({ limit: 5 })
  });

  if (isLoading) {
    return (
      <div className="space-y-3" data-testid="recent-tasks-loading">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-14 animate-pulse rounded-lg bg-white/5" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div
        data-testid="recent-tasks-error"
        className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-400"
      >
        <div className="flex items-center gap-2">
          <AlertCircle className="h-4 w-4" />
          <span>Failed to load recent tasks</span>
        </div>
      </div>
    );
  }

  const tasks = data?.data ?? [];

  if (tasks.length === 0) {
    return (
      <div
        data-testid="recent-tasks-empty"
        className="rounded-lg border border-white/10 bg-white/5 p-6 text-center"
      >
        <ListTodo className="mx-auto h-8 w-8 text-slate-500" />
        <p className="mt-2 text-slate-400">No tasks yet</p>
        <Link to="/tasks">
          <Button variant="outline" size="sm" className="mt-3">
            Create your first task
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-2" data-testid="recent-tasks-list">
      {tasks.map((task) => (
        <div
          key={task.id}
          data-testid={`task-item-${task.id}`}
          className="flex items-center justify-between rounded-lg border border-white/10 bg-white/5 p-3"
        >
          <div className="flex items-center gap-3">
            {task.status === "completed" ? (
              <CheckCircle2 className="h-4 w-4 text-green-400" />
            ) : (
              <Clock className="h-4 w-4 text-slate-400" />
            )}
            <span className={task.status === "completed" ? "text-slate-500 line-through" : ""}>
              {task.title}
            </span>
          </div>
          <span className="text-xs text-slate-500 capitalize">{task.status.replace("_", " ")}</span>
        </div>
      ))}
      <Link to="/tasks" className="block">
        <Button variant="outline" size="sm" className="mt-2 w-full">
          View all tasks
        </Button>
      </Link>
    </div>
  );
}

function DashboardContent() {
  const { data: tasksData, isLoading: tasksLoading } = useQuery({
    queryKey: ["tasks", {}],
    queryFn: () => fetchTasks({ limit: 100 })
  });

  const { data: projectsData, isLoading: projectsLoading } = useQuery({
    queryKey: ["projects", {}],
    queryFn: () => fetchProjects({ limit: 100 })
  });

  const tasks = tasksData?.data ?? [];
  const pendingTasks = tasks.filter((t) => t.status === "pending").length;
  const completedTasks = tasks.filter((t) => t.status === "completed").length;
  const totalProjects = projectsData?.pagination.total ?? 0;

  return (
    <div className="space-y-8" data-testid="dashboard-page">
      <div>
        <h2 className="text-2xl font-semibold">Dashboard</h2>
        <p className="mt-1 text-slate-400">Overview of your tasks and projects</p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard icon={ListTodo} label="Total Tasks" value={tasks.length} loading={tasksLoading} />
        <StatCard icon={Clock} label="Pending" value={pendingTasks} loading={tasksLoading} />
        <StatCard icon={CheckCircle2} label="Completed" value={completedTasks} loading={tasksLoading} />
        <StatCard icon={FolderKanban} label="Projects" value={totalProjects} loading={projectsLoading} />
      </div>

      {/* Recent Tasks Section */}
      <div>
        <h3 className="mb-4 text-lg font-medium">Recent Tasks</h3>
        <RecentTasks />
      </div>

      {/* Quick Actions */}
      <div>
        <h3 className="mb-4 text-lg font-medium">Quick Actions</h3>
        <div className="flex gap-3">
          <Link to="/tasks">
            <Button data-testid="quick-action-new-task">
              <ListTodo className="mr-2 h-4 w-4" />
              New Task
            </Button>
          </Link>
          <Link to="/projects">
            <Button variant="outline" data-testid="quick-action-new-project">
              <FolderKanban className="mr-2 h-4 w-4" />
              New Project
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}

export function Dashboard() {
  return (
    <ErrorBoundary>
      <DashboardContent />
    </ErrorBoundary>
  );
}
