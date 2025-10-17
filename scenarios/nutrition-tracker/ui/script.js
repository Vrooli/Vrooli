import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

// Nutrition Tracker UI JavaScript

if (typeof window !== 'undefined' && window.parent !== window && !window.__nutritionTrackerBridgeInitialized) {
    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[NutritionTracker] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'nutrition-tracker' });
    window.__nutritionTrackerBridgeInitialized = true;
}

// API endpoints
const API_BASE = 'http://localhost:8081/api';
const N8N_BASE = 'http://localhost:5678/webhook';

// State management
let currentMealType = 'breakfast';
let todaysMeals = [];
let userGoals = {
    calories: 2000,
    protein: 75,
    carbs: 225,
    fat: 65
};
let consumed = {
    calories: 0,
    protein: 0,
    carbs: 0,
    fat: 0
};

// Initialize the app
document.addEventListener('DOMContentLoaded', () => {
    initializeMealTypeButtons();
    loadTodaysMeals();
    updateProgressRings();
    setupAutoComplete();
});

// Meal type selector
function initializeMealTypeButtons() {
    const buttons = document.querySelectorAll('.meal-type-btn');
    buttons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            buttons.forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            currentMealType = e.target.dataset.meal;
        });
    });
}

// Analyze food using AI
async function analyzeFood() {
    const foodInput = document.getElementById('food-description');
    const description = foodInput.value.trim();
    
    if (!description) {
        showNotification('Please enter a food description', 'error');
        return;
    }
    
    showLoadingState(true);
    
    try {
        const response = await fetch(`${N8N_BASE}/nutrition-analyzer`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                food_description: description,
                meal_type: currentMealType,
                user_id: getUserId()
            })
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            addMealToUI(description, data);
            updateConsumedNutrients(data);
            updateProgressRings();
            foodInput.value = '';
            showNotification('Food added successfully!', 'success');
        }
    } catch (error) {
        console.error('Error analyzing food:', error);
        showNotification('Failed to analyze food. Please try again.', 'error');
    } finally {
        showLoadingState(false);
    }
}

// Load today's meals
async function loadTodaysMeals() {
    try {
        const response = await fetch(`${API_BASE}/meals/today?user_id=${getUserId()}`);
        const meals = await response.json();
        
        todaysMeals = meals;
        displayMeals(meals);
        calculateConsumedNutrients(meals);
    } catch (error) {
        console.error('Error loading meals:', error);
    }
}

// Display meals in the grid
function displayMeals(meals) {
    const mealsGrid = document.querySelector('.meals-grid');
    mealsGrid.innerHTML = '';
    
    meals.forEach(meal => {
        const mealCard = createMealCard(meal);
        mealsGrid.appendChild(mealCard);
    });
}

// Create meal card element
function createMealCard(meal) {
    const card = document.createElement('div');
    card.className = 'meal-card';
    card.innerHTML = `
        <div class="meal-emoji">${getMealEmoji(meal.meal_type)}</div>
        <div class="meal-name">${meal.name || meal.food_description}</div>
        <div class="meal-calories">${meal.calories} calories • ${capitalizeFirst(meal.meal_type)}</div>
    `;
    card.addEventListener('click', () => showMealDetails(meal));
    return card;
}

// Get emoji for meal type
function getMealEmoji(type) {
    const emojis = {
        breakfast: '🍳',
        lunch: '🥗',
        dinner: '🍝',
        snack: '🍎'
    };
    return emojis[type] || '🍽️';
}

// Update progress rings
function updateProgressRings() {
    updateProgressRing('calories', consumed.calories, userGoals.calories);
    updateProgressRing('protein', consumed.protein, userGoals.protein);
    updateProgressRing('carbs', consumed.carbs, userGoals.carbs);
}

// Update individual progress ring
function updateProgressRing(type, current, goal) {
    const percentage = Math.min((current / goal) * 100, 100);
    const circumference = 2 * Math.PI * 54; // radius = 54
    const offset = circumference - (percentage / 100) * circumference;
    
    const circle = document.querySelector(`.${type}-card circle:nth-child(2)`);
    if (circle) {
        circle.style.strokeDashoffset = offset;
    }
    
    // Update text values
    const valueElement = document.querySelector(`.${type}-card .progress-value`);
    const labelElement = document.querySelector(`.${type}-card .progress-label`);
    
    if (valueElement && labelElement) {
        if (type === 'calories') {
            valueElement.textContent = Math.round(current);
            labelElement.textContent = `of ${goal} kcal`;
        } else {
            valueElement.textContent = `${Math.round(current)}g`;
            labelElement.textContent = `of ${goal}g goal`;
        }
    }
}

// Calculate consumed nutrients from meals
function calculateConsumedNutrients(meals) {
    consumed = meals.reduce((acc, meal) => {
        acc.calories += meal.calories || 0;
        acc.protein += meal.protein || 0;
        acc.carbs += meal.carbs || 0;
        acc.fat += meal.fat || 0;
        return acc;
    }, { calories: 0, protein: 0, carbs: 0, fat: 0 });
}

// Update consumed nutrients after adding food
function updateConsumedNutrients(foodData) {
    consumed.calories += foodData.total_calories || 0;
    consumed.protein += foodData.protein_grams || 0;
    consumed.carbs += foodData.carbs_grams || 0;
    consumed.fat += foodData.fat_grams || 0;
}

// Add meal to UI
function addMealToUI(description, data) {
    const meal = {
        food_description: description,
        meal_type: currentMealType,
        calories: data.total_calories,
        protein: data.protein_grams,
        carbs: data.carbs_grams,
        fat: data.fat_grams
    };
    
    todaysMeals.push(meal);
    
    const mealsGrid = document.querySelector('.meals-grid');
    const mealCard = createMealCard(meal);
    mealsGrid.appendChild(mealCard);
}

// Toggle suggestions panel
function toggleSuggestions() {
    const panel = document.getElementById('suggestionsPanel');
    panel.classList.toggle('open');
    
    if (panel.classList.contains('open')) {
        loadMealSuggestions();
    }
}

// Load meal suggestions
async function loadMealSuggestions() {
    try {
        const remaining = {
            calories: userGoals.calories - consumed.calories,
            protein: userGoals.protein - consumed.protein,
            carbs: userGoals.carbs - consumed.carbs,
            fat: userGoals.fat - consumed.fat
        };
        
        const response = await fetch(`${N8N_BASE}/meal-suggester`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                user_id: getUserId(),
                meal_type: getNextMealType(),
                remaining_calories: remaining.calories,
                remaining_protein: remaining.protein,
                remaining_carbs: remaining.carbs,
                remaining_fat: remaining.fat
            })
        });
        
        const data = await response.json();
        displaySuggestions(data.suggestions);
    } catch (error) {
        console.error('Error loading suggestions:', error);
    }
}

// Display suggestions in panel
function displaySuggestions(suggestions) {
    const panel = document.getElementById('suggestionsPanel');
    const container = panel.querySelector('.suggestions-header').nextElementSibling;
    
    // Clear existing suggestions
    while (container) {
        container.remove();
    }
    
    // Add new suggestions
    suggestions.forEach(suggestion => {
        const item = document.createElement('div');
        item.className = 'suggestion-item';
        item.innerHTML = `
            <h3>${suggestion.meal_name}</h3>
            <p style="color: var(--text-light); margin-top: 5px;">
                ${suggestion.calories} cal • ${suggestion.protein}g protein
            </p>
            <p style="color: var(--primary-green); margin-top: 10px; font-size: 14px;">
                ✓ ${suggestion.recommendation_reason}
            </p>
        `;
        item.addEventListener('click', () => acceptSuggestion(suggestion));
        panel.appendChild(item);
    });
}

// Accept a meal suggestion
function acceptSuggestion(suggestion) {
    const foodInput = document.getElementById('food-description');
    foodInput.value = suggestion.meal_name;
    toggleSuggestions();
    showNotification('Suggestion added to input. Click "Add Food" to log it!', 'info');
}

// Get next meal type based on time
function getNextMealType() {
    const hour = new Date().getHours();
    if (hour < 11) return 'breakfast';
    if (hour < 15) return 'lunch';
    if (hour < 20) return 'dinner';
    return 'snack';
}

// Setup autocomplete for food input
function setupAutoComplete() {
    const input = document.getElementById('food-description');
    let timeout;
    
    input.addEventListener('input', (e) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => {
            if (e.target.value.length > 2) {
                searchFoods(e.target.value);
            }
        }, 300);
    });
}

// Search foods in database
async function searchFoods(query) {
    try {
        const response = await fetch(`${API_BASE}/foods/search?q=${encodeURIComponent(query)}`);
        const foods = await response.json();
        // Implement autocomplete dropdown here if needed
    } catch (error) {
        console.error('Error searching foods:', error);
    }
}

// Show notification
function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 15px 25px;
        background: ${type === 'success' ? 'var(--primary-green)' : type === 'error' ? '#f44336' : '#2196F3'};
        color: white;
        border-radius: 10px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 10000;
        animation: slideIn 0.3s ease;
    `;
    notification.textContent = message;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Show loading state
function showLoadingState(loading) {
    const addBtn = document.querySelector('.add-food-btn');
    if (loading) {
        addBtn.textContent = 'Analyzing...';
        addBtn.disabled = true;
    } else {
        addBtn.textContent = '+ Add Food';
        addBtn.disabled = false;
    }
}

// Get user ID (mock for demo)
function getUserId() {
    return localStorage.getItem('userId') || 'demo-user-123';
}

// Capitalize first letter
function capitalizeFirst(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

// Show meal details modal
function showMealDetails(meal) {
    // Implement meal details modal here
    console.log('Show details for:', meal);
}

// Add animations CSS
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
`;
document.head.appendChild(style);
