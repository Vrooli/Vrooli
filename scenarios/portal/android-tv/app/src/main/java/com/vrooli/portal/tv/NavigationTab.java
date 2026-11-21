package com.vrooli.portal.tv;
import android.content.Context;
import android.graphics.Canvas;
import android.graphics.Paint;
import android.widget.TextView;
/** Selection remains visible independently of the single remote focus. */
final class NavigationTab extends TextView {
    private final Paint underline = new Paint(Paint.ANTI_ALIAS_FLAG);
    NavigationTab(Context context, String label) {
        super(context); setText(label); setTextSize(18); setTextColor(TvTheme.SECONDARY); setIncludeFontPadding(false);
        underline.setColor(TvTheme.ACCENT);
    }
    @Override protected void onDraw(Canvas canvas) {
        super.onDraw(canvas);
        if (isSelected()) {
            float inset = TvTheme.dp(getContext(), 8), y = getHeight() - TvTheme.dp(getContext(), 2);
            canvas.drawRoundRect(inset, y, getWidth() - inset, getHeight(), 2, 2, underline);
        }
    }
}
