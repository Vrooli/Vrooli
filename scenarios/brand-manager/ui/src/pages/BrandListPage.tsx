import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Search, RefreshCw } from "lucide-react";
import { fetchBrands } from "../lib/api";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { ErrorAlert } from "../components/ui/error-alert";
import { BrandCard } from "../components/brand-card";

// [REQ:BM-REQ-UI-LIBRARY] [REQ:BM-REQ-UI-DASHBOARD]

interface BrandListPageProps {
  onNavigate: (path: string) => void;
}

export default function BrandListPage({ onNavigate }: BrandListPageProps) {
  const [search, setSearch] = useState("");

  const { data: brands, isLoading, error, refetch } = useQuery({
    queryKey: ["brands", search],
    queryFn: () => fetchBrands(search || undefined),
  });

  return (
    <div data-testid="brand-list-page">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-slate-50">Brand Library</h1>
        <Button onClick={() => onNavigate("/brands/new")} data-testid="create-brand-btn">
          <Plus className="mr-2 h-4 w-4" />
          New Brand
        </Button>
      </div>

      <div className="flex items-center gap-3 mb-6">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <Input
            variant="search"
            placeholder="Search brands..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            data-testid="brand-search-input"
          />
        </div>
        <button
          onClick={() => refetch()}
          className="rounded-lg border border-white/10 bg-white/5 p-2 text-slate-400 hover:text-slate-50 hover:bg-white/10 transition-colors"
          data-testid="refresh-brands-btn"
        >
          <RefreshCw className="h-4 w-4" />
        </button>
      </div>

      {isLoading && (
        <div className="text-center text-slate-400 py-12">Loading brands...</div>
      )}

      {error && (
        <ErrorAlert
          error={error}
          fallbackMessage="Unable to load brands."
          fallbackRecovery="Make sure the API is running."
          onRetry={() => refetch()}
          testId="brand-list-error"
        />
      )}

      {brands && brands.length === 0 && (
        <div className="text-center py-12" data-testid="brand-list-empty">
          <p className="text-slate-400 mb-4">No brands found. Create your first brand to get started.</p>
          <Button variant="outline" onClick={() => onNavigate("/brands/new")}>
            <Plus className="mr-2 h-4 w-4" />
            Create Brand
          </Button>
        </div>
      )}

      {brands && brands.length > 0 && (
        <div className="grid gap-4" data-testid="brand-list-grid">
          {brands.map((brand) => (
            <BrandCard
              key={brand.id}
              brand={brand}
              onClick={() => onNavigate(`/brands/${brand.id}`)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
