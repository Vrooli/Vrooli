import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { ArrowLeft, ArrowRight, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Badge } from "../components/ui/badge";
import { createProfile } from "../lib/api";

const TIER_INFO = [
  { id: 1, name: "Local/Dev", description: "Full Vrooli installation for development" },
  { id: 2, name: "Desktop", description: "Windows, macOS, Linux standalone applications" },
  { id: 3, name: "Mobile", description: "iOS and Android native apps" },
  { id: 4, name: "SaaS/Cloud", description: "DigitalOcean, AWS, bare metal deployments" },
  { id: 5, name: "Enterprise", description: "Hardware appliances with compliance" },
];

export function NewProfile() {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({
    name: "",
    scenario: "",
    tiers: [] as number[],
  });

  const createMutation = useMutation({
    mutationFn: createProfile,
    onSuccess: (data) => {
      navigate(`/profiles/${data.id}`);
    },
  });

  const handleTierToggle = (tierId: number) => {
    setFormData((prev) => ({
      ...prev,
      tiers: prev.tiers.includes(tierId)
        ? prev.tiers.filter((t) => t !== tierId)
        : [...prev.tiers, tierId],
    }));
  };

  const handleSubmit = () => {
    createMutation.mutate(formData);
  };

  const canProceed = () => {
    if (step === 1) return formData.name && formData.scenario;
    if (step === 2) return formData.tiers.length > 0;
    return true;
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="outline"
          size="icon"
          onClick={() => navigate(-1)}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold">Create Deployment Profile</h1>
          <p className="text-slate-400 mt-1">
            Step {step} of 3
          </p>
        </div>
      </div>

      {/* Progress */}
      <div className="flex gap-2">
        {[1, 2, 3].map((s) => (
          <div
            key={s}
            className={`h-1 flex-1 rounded-full ${
              s <= step ? "bg-cyan-500" : "bg-slate-800"
            }`}
          />
        ))}
      </div>

      {/* Step 1: Basic Info */}
      {step === 1 && (
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
            <CardDescription>
              Name your deployment profile and select the scenario to deploy
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Profile Name</Label>
              <Input
                id="name"
                placeholder="e.g., picker-wheel-desktop"
                value={formData.name}
                onChange={(e) =>
                  setFormData((prev) => ({ ...prev, name: e.target.value }))
                }
              />
              <p className="text-xs text-slate-400">
                A descriptive name for this deployment configuration
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="scenario">Scenario</Label>
              <Input
                id="scenario"
                placeholder="e.g., picker-wheel"
                value={formData.scenario}
                onChange={(e) =>
                  setFormData((prev) => ({ ...prev, scenario: e.target.value }))
                }
              />
              <p className="text-xs text-slate-400">
                The scenario you want to deploy
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 2: Select Tiers */}
      {step === 2 && (
        <Card>
          <CardHeader>
            <CardTitle>Target Deployment Tiers</CardTitle>
            <CardDescription>
              Select which platforms you want to deploy to
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {TIER_INFO.map((tier) => (
              <button
                key={tier.id}
                onClick={() => handleTierToggle(tier.id)}
                className={`w-full text-left rounded-lg border p-4 transition-colors ${
                  formData.tiers.includes(tier.id)
                    ? "border-cyan-500 bg-cyan-500/10"
                    : "border-white/10 bg-white/5 hover:bg-white/10"
                }`}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-medium">Tier {tier.id}: {tier.name}</h3>
                      {formData.tiers.includes(tier.id) && (
                        <Badge variant="success">Selected</Badge>
                      )}
                    </div>
                    <p className="text-sm text-slate-400 mt-1">
                      {tier.description}
                    </p>
                  </div>
                </div>
              </button>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Step 3: Review */}
      {step === 3 && (
        <Card>
          <CardHeader>
            <CardTitle>Review & Create</CardTitle>
            <CardDescription>
              Review your deployment profile configuration
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label className="text-slate-400">Profile Name</Label>
              <p className="text-lg font-medium mt-1">{formData.name}</p>
            </div>

            <div>
              <Label className="text-slate-400">Scenario</Label>
              <p className="text-lg font-medium mt-1">{formData.scenario}</p>
            </div>

            <div>
              <Label className="text-slate-400">Target Tiers</Label>
              <div className="flex flex-wrap gap-2 mt-2">
                {formData.tiers.map((tierId) => {
                  const tier = TIER_INFO.find((t) => t.id === tierId);
                  return (
                    <Badge key={tierId} variant="secondary">
                      Tier {tierId}: {tier?.name}
                    </Badge>
                  );
                })}
              </div>
            </div>

            {createMutation.isError && (
              <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-200">
                <p className="text-sm">
                  Failed to create profile: {(createMutation.error as Error).message}
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Navigation */}
      <div className="flex items-center justify-between">
        <Button
          variant="outline"
          onClick={() => setStep((s) => Math.max(1, s - 1))}
          disabled={step === 1}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Previous
        </Button>

        {step < 3 ? (
          <Button
            onClick={() => setStep((s) => s + 1)}
            disabled={!canProceed()}
          >
            Next
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        ) : (
          <Button
            onClick={handleSubmit}
            disabled={createMutation.isPending || !canProceed()}
          >
            {createMutation.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Create Profile
          </Button>
        )}
      </div>
    </div>
  );
}
