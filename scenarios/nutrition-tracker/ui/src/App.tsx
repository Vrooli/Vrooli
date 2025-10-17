import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Apple,
  Beef,
  ChefHat,
  FlaskConical,
  Flame,
  Layers3,
  Loader2,
  MoonStar,
  Plus,
  RefreshCw,
  Sparkles,
  Sunrise,
  UtensilsCrossed
} from 'lucide-react';

import { ensureIframeBridge } from './lib/bridge';
import { resolveEndpointBases, type EndpointBases } from './lib/proxy';
import { cn } from './lib/utils';
import { AnalyzerResponse, Meal, MealType, NutritionGoals, Suggestion } from './types';

import { Badge } from './components/ui/badge';
import { Button } from './components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from './components/ui/card';
import { Progress } from './components/ui/progress';
import { ScrollArea } from './components/ui/scroll-area';
import { Tabs, TabsList, TabsTrigger } from './components/ui/tabs';
import { Textarea } from './components/ui/textarea';
import { Toaster } from './components/ui/toaster';
import { useToast } from './components/ui/use-toast';

const DEFAULT_GOALS: NutritionGoals = {
  calories: 2000,
  protein: 75,
  carbs: 225,
  fat: 65
};

const MEAL_TYPES: Array<{
  value: MealType;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  description: string;
}> = [
  {
    value: 'breakfast',
    label: 'Breakfast',
    icon: Sunrise,
    description: 'Start strong with high-protein fuel.'
  },
  {
    value: 'lunch',
    label: 'Lunch',
    icon: UtensilsCrossed,
    description: 'Balanced midday energy boost.'
  },
  {
    value: 'dinner',
    label: 'Dinner',
    icon: MoonStar,
    description: 'Wind down with nourishing meals.'
  },
  {
    value: 'snack',
    label: 'Snack',
    icon: Apple,
    description: 'Smart bites between meals.'
  }
];

const MEAL_TYPE_COLORS: Record<MealType, string> = {
  breakfast: 'from-amber-200 to-amber-100 text-amber-800',
  lunch: 'from-sky-200 to-blue-100 text-sky-900',
  dinner: 'from-indigo-200 to-indigo-100 text-indigo-900',
  snack: 'from-emerald-200 to-emerald-100 text-emerald-900'
};

const MACRO_METADATA: Array<{
  key: keyof NutritionGoals;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  accent: string;
  unit: string;
}> = [
  { key: 'calories', label: 'Calories', icon: Flame, accent: 'from-rose-500 to-orange-500', unit: 'kcal' },
  { key: 'protein', label: 'Protein', icon: Beef, accent: 'from-emerald-500 to-teal-500', unit: 'g' },
  { key: 'carbs', label: 'Carbs', icon: Layers3, accent: 'from-sky-500 to-cyan-500', unit: 'g' },
  { key: 'fat', label: 'Fats', icon: FlaskConical, accent: 'from-purple-500 to-fuchsia-500', unit: 'g' }
];

const SESSION_USER_KEY = 'nutrition-tracker-user-id';

function getSessionUserId() {
  if (typeof window === 'undefined') {
    return 'demo-user-123';
  }
  const existing = window.localStorage.getItem(SESSION_USER_KEY);
  if (existing) {
    return existing;
  }
  const generated = `demo-user-${Math.random().toString(36).slice(2, 8)}`;
  window.localStorage.setItem(SESSION_USER_KEY, generated);
  return generated;
}

function App() {
  const [goals, setGoals] = useState<NutritionGoals>(DEFAULT_GOALS);
  const [meals, setMeals] = useState<Meal[]>([]);
  const [mealType, setMealType] = useState<MealType>('breakfast');
  const [isLoading, setIsLoading] = useState(true);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [isSuggestionsLoading, setIsSuggestionsLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [endpoints, setEndpoints] = useState(() => resolveEndpointBases());

  const descriptionRef = useRef<HTMLTextAreaElement | null>(null);
  const { toast } = useToast();

  const userId = useMemo(() => getSessionUserId(), []);

  const loadInitialData = useCallback(
    async (bases: EndpointBases) => {
      setIsLoading(true);
      try {
        const [goalResponse, mealsResponse] = await Promise.all([
          fetch(`${bases.apiBase}/goals?user_id=${userId}`),
          fetch(`${bases.apiBase}/meals/today?user_id=${userId}`)
        ]);

        if (goalResponse.ok) {
          const goalPayload = await goalResponse.json();
          setGoals((prev) => ({
            calories: Number(goalPayload?.daily_calories) || prev.calories,
            protein: Number(goalPayload?.protein_grams) || prev.protein,
            carbs: Number(goalPayload?.carbs_grams) || prev.carbs,
            fat: Number(goalPayload?.fat_grams) || prev.fat
          }));
        }

        if (mealsResponse.ok) {
          const mealsPayload = await mealsResponse.json();
          if (Array.isArray(mealsPayload)) {
            setMeals(mealsPayload);
          }
        }
      } catch (error) {
        console.error('[NutritionTracker] Failed to load initial data', error);
        toast({
          variant: 'destructive',
          title: 'Unable to load data',
          description: 'Check your connection and try again.'
        });
      } finally {
        setIsLoading(false);
      }
    },
    [toast, userId]
  );

  useEffect(() => {
    ensureIframeBridge();
    const resolved = resolveEndpointBases();
    setEndpoints(resolved);
    void loadInitialData(resolved);
  }, [loadInitialData]);

  const totals = useMemo(() => {
    return meals.reduce(
      (acc, meal) => {
        acc.calories += Number(meal.calories) || 0;
        acc.protein += Number(meal.protein) || 0;
        acc.carbs += Number(meal.carbs) || 0;
        acc.fat += Number(meal.fat) || 0;
        return acc;
      },
      { calories: 0, protein: 0, carbs: 0, fat: 0 }
    );
  }, [meals]);

  const remaining = useMemo(() => ({
    calories: Math.max(0, goals.calories - totals.calories),
    protein: Math.max(0, goals.protein - totals.protein),
    carbs: Math.max(0, goals.carbs - totals.carbs),
    fat: Math.max(0, goals.fat - totals.fat)
  }), [goals, totals]);

  const refreshSuggestions = useCallback(async () => {
    setIsSuggestionsLoading(true);
    try {
      const remainingCalories = Math.max(0, goals.calories - totals.calories);
      const remainingProtein = Math.max(0, goals.protein - totals.protein);
      const remainingCarbs = Math.max(0, goals.carbs - totals.carbs);
      const remainingFat = Math.max(0, goals.fat - totals.fat);

      const response = await fetch(`${endpoints.n8nBase}/meal-suggester`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: userId,
          meal_type: inferNextMealType(),
          remaining_calories: remainingCalories,
          remaining_protein: remainingProtein,
          remaining_carbs: remainingCarbs,
          remaining_fat: remainingFat
        })
      });

      if (!response.ok) {
        throw new Error(`Status ${response.status}`);
      }

      const payload = (await response.json()) as { suggestions?: Suggestion[] };
      setSuggestions(Array.isArray(payload?.suggestions) ? payload.suggestions : []);
    } catch (error) {
      console.error('[NutritionTracker] Failed to load suggestions', error);
      toast({
        variant: 'destructive',
        title: 'Suggestions unavailable',
        description: 'AI meal ideas are temporarily unavailable.'
      });
      setSuggestions([]);
    } finally {
      setIsSuggestionsLoading(false);
    }
  }, [endpoints.n8nBase, goals.calories, goals.carbs, goals.fat, goals.protein, toast, totals.calories, totals.carbs, totals.fat, totals.protein, userId]);

  useEffect(() => {
    if (isLoading) {
      return;
    }
    void refreshSuggestions();
  }, [isLoading, refreshSuggestions]);

  async function analyzeFood() {
    const description = descriptionRef.current?.value.trim();
    if (!description) {
      toast({
        variant: 'destructive',
        title: 'Add a description first',
        description: 'Tell us what you ate so we can analyze it.'
      });
      return;
    }

    setIsAnalyzing(true);
    try {
      const response = await fetch(`${endpoints.n8nBase}/nutrition-analyzer`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          food_description: description,
          meal_type: mealType,
          user_id: userId
        })
      });
      if (!response.ok) {
        throw new Error(`Status ${response.status}`);
      }
      const data = (await response.json()) as AnalyzerResponse;
      if (data.status !== 'success') {
        throw new Error('Analyzer returned an error');
      }

      const normalizedMeal: Meal = {
        food_description: description,
        meal_type: mealType,
        calories: data.total_calories,
        protein: data.protein_grams,
        carbs: data.carbs_grams,
        fat: data.fat_grams
      };

      let persistedMeal: Meal | undefined;
      try {
        const saveResponse = await fetch(`${endpoints.apiBase}/meals`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            user_id: userId,
            meal_type: normalizedMeal.meal_type,
            meal_date: new Date().toISOString().slice(0, 10),
            food_description: normalizedMeal.food_description,
            calories: normalizedMeal.calories,
            protein: normalizedMeal.protein ?? 0,
            carbs: normalizedMeal.carbs ?? 0,
            fat: normalizedMeal.fat ?? 0,
            fiber: 0,
            sugar: 0,
            sodium: 0
          })
        });

        if (saveResponse.ok) {
          persistedMeal = (await saveResponse.json()) as Meal;
        } else {
          const message = `Status ${saveResponse.status}`;
          throw new Error(message);
        }
      } catch (persistError) {
        console.error('[NutritionTracker] Failed to persist meal', persistError);
        toast({
          variant: 'destructive',
          title: 'Meal saved locally only',
          description: 'API persistence failed—check the service before refreshing.'
        });
      }

      setMeals((prev) => [...prev, persistedMeal ?? normalizedMeal]);
      if (descriptionRef.current) {
        descriptionRef.current.value = '';
      }

      toast({
        variant: 'success',
        title: 'Meal logged',
        description: 'Great choice! Progress updated.'
      });
      void refreshSuggestions();
    } catch (error) {
      console.error('[NutritionTracker] Failed to analyze meal', error);
      toast({
        variant: 'destructive',
        title: 'Unable to analyze meal',
        description: 'Please try again in a moment.'
      });
    } finally {
      setIsAnalyzing(false);
    }
  }

  function inferNextMealType(): MealType {
    const hour = new Date().getHours();
    if (hour < 11) return 'breakfast';
    if (hour < 15) return 'lunch';
    if (hour < 20) return 'dinner';
    return 'snack';
  }

  const isEmpty = !isLoading && meals.length === 0;

  return (
    <div className="bg-hero-pattern">
      <div className="relative z-10 container space-y-8 pb-16 pt-12">
        <header className="relative flex flex-col gap-6 rounded-2xl border border-border/70 bg-white/80 p-8 shadow-sm backdrop-blur-md">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
            <div className="flex items-start gap-4">
              <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-brand-500 to-emerald-600 text-white shadow-lg">
                <ChefHat className="h-7 w-7" />
              </div>
              <div className="space-y-2">
                <h1 className="font-heading text-3xl font-semibold tracking-tight text-slate-900 lg:text-4xl">
                  Nutrition Tracker
                </h1>
                <p className="text-base text-slate-600 lg:text-lg">
                  A focused workspace for logging meals, understanding macros, and keeping your goals on track.
                </p>
              </div>
            </div>
            <div className="flex flex-wrap gap-3 text-sm">
              <Badge className="bg-brand-100 text-brand-700">{inferNextMealType().toUpperCase()} FOCUS</Badge>
              <Badge variant="muted">Daily goal: {goals.calories} kcal</Badge>
              <Badge variant="muted">Remaining protein: {Math.max(0, Math.round(remaining.protein))}g</Badge>
            </div>
          </div>
        </header>

        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {MACRO_METADATA.map(({ key, label, icon: Icon, accent, unit }) => {
            const target = goals[key];
            const current = totals[key];
            const percent = target === 0 ? 0 : Math.min(100, (current / target) * 100);
            return (
              <Card key={key} className="border-none bg-white/80 shadow-sm">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
                  <div className="space-y-1">
                    <CardDescription>{label}</CardDescription>
                    <CardTitle className="text-2xl font-semibold">
                      {Math.round(current)} <span className="text-sm font-medium text-muted-foreground">{unit}</span>
                    </CardTitle>
                  </div>
                  <div className={cn('flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br text-white shadow-sm', accent)}>
                    <Icon className="h-6 w-6" />
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  <Progress value={percent} />
                  <p className="text-xs text-muted-foreground">
                    {Math.max(0, Math.round(target - current))} {unit} remaining
                  </p>
                </CardContent>
              </Card>
            );
          })}
        </section>

        <div className="grid gap-6 lg:grid-cols-[1.05fr_0.95fr]">
          <Card className="border-none bg-white/85 shadow-md">
            <CardHeader className="space-y-2 pb-4">
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-2xl">Log what you just ate</CardTitle>
                  <CardDescription>AI will estimate macros instantly and record the meal for today.</CardDescription>
                </div>
              </div>
              <Tabs value={mealType} onValueChange={(value) => setMealType(value as MealType)}>
                <TabsList className="grid grid-cols-2 gap-2 bg-white p-1 shadow-inner sm:grid-cols-4">
                  {MEAL_TYPES.map(({ value, label, icon: Icon, description }) => (
                    <TabsTrigger key={value} value={value} className="h-auto flex-col items-start gap-1 px-4 py-3 text-left text-sm">
                      <div className="flex w-full items-center gap-2">
                        <div
                          className={cn(
                            'flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br shadow-sm',
                            MEAL_TYPE_COLORS[value]
                          )}
                        >
                          <Icon className="h-4 w-4" />
                        </div>
                        <span className="font-medium">{label}</span>
                      </div>
                      <span className="text-xs text-muted-foreground">{description}</span>
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            </CardHeader>
            <CardContent className="space-y-4">
              <Textarea
                ref={descriptionRef}
                placeholder="e.g. grilled salmon with quinoa, roasted veggies, and a lemon dressing"
                className="min-h-[120px] resize-none"
              />
              <div className="flex flex-col-reverse gap-3 sm:flex-row sm:items-center sm:justify-between">
                <p className="text-sm text-muted-foreground">
                  Your insights improve as you log consistently. Results sync instantly with the Nutrition Tracker API.
                </p>
                <Button onClick={analyzeFood} disabled={isAnalyzing} className="min-w-[150px]">
                  {isAnalyzing ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin" />
                      Analyzing...
                    </>
                  ) : (
                    <>
                      <Plus className="h-4 w-4" />
                      Log meal
                    </>
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card className="border-none bg-white/85 shadow-md">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
              <div>
                <CardTitle className="text-2xl">Today&apos;s meals</CardTitle>
                <CardDescription>{isEmpty ? 'Nothing logged yet. Your first meal sets the tone.' : 'Tap a card to review macro details.'}</CardDescription>
              </div>
              <Badge variant="muted">{meals.length} logged</Badge>
            </CardHeader>
            <CardContent className="h-[360px]">
              <ScrollArea className="h-full pr-2">
                {isLoading ? (
                  <div className="space-y-3">
                    {Array.from({ length: 4 }).map((_, index) => (
                      <div
                        key={index}
                        className="h-20 animate-pulse rounded-xl bg-gradient-to-r from-muted to-white/80"
                      />
                    ))}
                  </div>
                ) : isEmpty ? (
                  <div className="flex h-full flex-col items-center justify-center rounded-xl border border-dashed border-border/70 bg-muted/50 p-6 text-center">
                    <Sparkles className="mb-3 h-6 w-6 text-brand-500" />
                    <p className="font-medium text-slate-700">No meals yet</p>
                    <p className="text-sm text-muted-foreground">
                      Capture your first meal to see macros and personalized guidance flow in.
                    </p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {meals.map((meal, index) => (
                      <article
                        key={`${meal.food_description ?? meal.name ?? 'meal'}-${index}`}
                        className="group flex items-start gap-3 rounded-xl border border-border/70 bg-white/95 p-4 shadow-sm transition hover:-translate-y-0.5 hover:border-brand-200 hover:shadow-lg"
                      >
                        <div
                          className={cn(
                            'mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br text-sm font-semibold text-white shadow-sm',
                            MEAL_TYPE_COLORS[meal.meal_type]
                          )}
                        >
                          <UtensilsCrossed className="h-5 w-5 text-slate-900/90" />
                        </div>
                        <div className="space-y-1">
                          <p className="font-medium text-slate-900">
                            {meal.name || meal.food_description || 'Meal'}
                          </p>
                          <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
                            <span>{Math.round(Number(meal.calories) || 0)} kcal</span>
                            <span className="h-1.5 w-1.5 rounded-full bg-border" />
                            <span>{Math.round(Number(meal.protein) || 0)}g protein</span>
                            <span className="h-1.5 w-1.5 rounded-full bg-border" />
                            <span className="capitalize">{meal.meal_type}</span>
                          </div>
                        </div>
                      </article>
                    ))}
                  </div>
                )}
              </ScrollArea>
            </CardContent>
          </Card>
        </div>

        <div className="grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
          <Card className="border-none bg-white/85 shadow-md">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
              <div>
                <CardTitle className="text-2xl">AI meal suggestions</CardTitle>
                <CardDescription>
                  Curated ideas that match your remaining macros and preferences.
                </CardDescription>
              </div>
              <Button
                variant="secondary"
                size="sm"
                onClick={refreshSuggestions}
                disabled={isSuggestionsLoading}
                className="min-w-[120px]"
              >
                {isSuggestionsLoading ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Updating
                  </>
                ) : (
                  <>
                    <RefreshCw className="h-4 w-4" />
                    Refresh
                  </>
                )}
              </Button>
            </CardHeader>
            <CardContent>
              {isSuggestionsLoading ? (
                <div className="space-y-3">
                  {Array.from({ length: 3 }).map((_, index) => (
                    <div
                      key={index}
                      className="h-24 animate-pulse rounded-xl bg-gradient-to-r from-muted to-white/90"
                    />
                  ))}
                </div>
              ) : suggestions.length === 0 ? (
                <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border/70 bg-muted/40 p-6 text-center">
                  <Sparkles className="mb-3 h-6 w-6 text-brand-500" />
                  <p className="font-medium text-slate-700">No suggestions yet</p>
                  <p className="text-sm text-muted-foreground">
                    Log a meal or refresh to receive tailored inspiration.
                  </p>
                </div>
              ) : (
                <div className="space-y-4">
                  {suggestions.map((suggestion, index) => {
                    const calories = Math.round(suggestion.calories ?? suggestion.total_calories ?? 0);
                    const protein = Math.round(suggestion.protein ?? suggestion.protein_grams ?? 0);
                    return (
                      <div
                        key={`${suggestion.meal_name}-${index}`}
                        className="flex flex-col gap-3 rounded-xl border border-border/60 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-lg"
                      >
                        <div className="flex items-start justify-between gap-4">
                          <div>
                            <h3 className="text-lg font-semibold text-slate-900">{suggestion.meal_name}</h3>
                            <p className="mt-1 text-sm text-muted-foreground">
                              {calories} kcal · {protein}g protein
                            </p>
                          </div>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              if (descriptionRef.current) {
                                descriptionRef.current.value = suggestion.meal_name;
                                descriptionRef.current.focus();
                              }
                              toast({
                                title: 'Suggestion copied',
                                description: 'Press “Log meal” to analyze this idea.'
                              });
                            }}
                          >
                            <Plus className="h-4 w-4" />
                            Use idea
                          </Button>
                        </div>
                        <p className="text-sm text-brand-700">
                          <Sparkles className="mr-2 inline h-4 w-4" />
                          {suggestion.recommendation_reason || 'Balanced for your goals today.'}
                        </p>
                      </div>
                    );
                  })}
                </div>
              )}
            </CardContent>
          </Card>

          <Card className="border-none bg-white/85 shadow-md">
            <CardHeader className="space-y-2 pb-4">
              <CardTitle className="text-2xl">Goal alignment</CardTitle>
              <CardDescription>
                Track how today’s intake compares to your targets and adjust in real time.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                {MACRO_METADATA.map(({ key, label, unit }) => {
                  const consumed = totals[key];
                  const target = goals[key];
                  const percentage = target === 0 ? 0 : Math.min(100, (consumed / target) * 100);
                  const deficit = Math.max(0, target - consumed);
                  return (
                    <div
                      key={key}
                      className="rounded-xl border border-border/70 bg-white/95 p-4 shadow-sm"
                    >
                      <div className="flex items-center justify-between">
                        <p className="text-sm font-medium text-slate-700">{label}</p>
                        <span className="text-xs text-muted-foreground">Target {target} {unit}</span>
                      </div>
                      <div className="mt-3 space-y-2">
                        <Progress value={percentage} />
                        <div className="flex items-center justify-between text-sm text-muted-foreground">
                          <span>{Math.round(consumed)} {unit}</span>
                          <span>{Math.round(deficit)} {unit} remaining</span>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
              <div className="rounded-xl border border-dashed border-brand-200 bg-brand-50/70 p-5 text-sm text-brand-900">
                <p className="font-medium text-brand-800">Pro tip</p>
                <p className="mt-1 leading-relaxed">
                  Keep protein intake steady across meals. When macros get close to goal, refresh suggestions to find lightweight options that fill the exact gap.
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <Toaster />
    </div>
  );
}

export default App;
