package com.vrooli.portal.tv;

import android.animation.ValueAnimator;
import android.content.Context;
import android.graphics.Color;
import android.graphics.Typeface;
import android.graphics.drawable.GradientDrawable;
import android.view.Gravity;
import android.view.View;
import android.widget.LinearLayout;
import android.widget.TextView;

/** Native TV tokens and focus primitives. Layouts compose these without owning colors. */
final class TvTheme {
    static final int INK = 0xFF081320;
    static final int TEXT = 0xFFF5F5F2;
    static final int SECONDARY = 0xFFBBC9D9;
    static final int ACCENT = 0xFFAFF0D0;
    static final int SURFACE = 0xE3102033;
    static final int FOCUS = 0xFFF4EFDE;
    static int dp(Context context, float value) {
        return Math.round(value * context.getResources().getDisplayMetrics().density);
    }
    static TextView text(Context context, String value, int size, int color) {
        TextView view = new TextView(context);
        view.setText(value);
        view.setTextColor(color);
        view.setTextSize(size);
        view.setFontFeatureSettings("kern");
        view.setIncludeFontPadding(false);
        view.setTypeface(Typeface.create("sans-serif", Typeface.NORMAL));
        return view;
    }
    static GradientDrawable surface(Context context, int color, boolean focused) {
        GradientDrawable result = new GradientDrawable();
        result.setColor(color);
        result.setCornerRadius(dp(context, 10));
        result.setStroke(dp(context, focused ? 2 : 1), focused ? FOCUS : 0x194B647F);
        return result;
    }
    static LinearLayout column(Context context) {
        LinearLayout result = new LinearLayout(context);
        result.setOrientation(LinearLayout.VERTICAL);
        result.setClipChildren(false);
        result.setClipToPadding(false);
        return result;
    }
    static TextView action(Context context, String label, Runnable action) {
        TextView result = text(context, label, 16, TEXT);
        result.setGravity(Gravity.CENTER);
        result.setPadding(dp(context, 16), dp(context, 9), dp(context, 16), dp(context, 9));
        result.setFocusable(true);
        result.setClickable(true);
        result.setBackground(surface(context, SURFACE, false));
        result.setOnFocusChangeListener((v, focused) -> {
            v.setBackground(surface(context, focused ? 0xFF24394F : SURFACE, focused));
        });
        result.setOnClickListener(v -> action.run());
        return result;
    }
    static void focus(View view, boolean focused) {
        view.setBackground(surface(view.getContext(), focused ? 0xF025394D : SURFACE, focused));
        view.animate().cancel();
        // System reduced-motion preference is authoritative; an outline is always immediate.
        float scale = focused && ValueAnimator.areAnimatorsEnabled() ? 1.025f : 1f;
        if (ValueAnimator.areAnimatorsEnabled()) view.animate().scaleX(scale).scaleY(scale).setDuration(100).start();
        else { view.setScaleX(scale); view.setScaleY(scale); }
        view.setElevation(dp(view.getContext(), focused ? 4 : 0));
    }
    private TvTheme() {}
}
