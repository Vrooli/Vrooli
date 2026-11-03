package com.vrooli.devicecontrol.capture;

import android.app.Activity;
import android.Manifest;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.media.projection.MediaProjectionManager;
import android.os.Build;
import android.os.Bundle;
import android.os.Environment;
import android.widget.TextView;

import java.io.File;
import java.util.regex.Pattern;

/** The Android-owned consent boundary for one playback-audio capture session. */
public final class CaptureActivity extends Activity {
    public static final String ACTION_STOP = "com.vrooli.devicecontrol.capture.STOP";
    static final String EXTRA_SESSION = "session_id";
    private static final int REQUEST_RECORD_AUDIO = 40;
    private static final int REQUEST_PROJECTION = 41;
    private static final Pattern SESSION_ID = Pattern.compile("[a-zA-Z0-9-]{1,80}");

    private String sessionId;
    private boolean projectionRequested;

    @Override protected void onCreate(Bundle state) {
        super.onCreate(state);
        sessionId = getIntent().getStringExtra(EXTRA_SESSION);
        if (ACTION_STOP.equals(getIntent().getAction())) {
            stopSession();
            return;
        }
        if (sessionId == null || !SESSION_ID.matcher(sessionId).matches()) {
            finish();
            return;
        }
        status("awaiting_audio_permission", "Android audio permission and screen-capture approval are required for playback capture.");
        TextView message = new TextView(this);
        message.setText("Record device playback\n\n" +
                "Android needs audio permission and screen-capture approval for playback capture. " +
                "The audio permission is requested only if it has not already been granted.\n\n" +
                "1. Allow audio permission if Android asks. Android requires it for playback capture. " +
                "Device Control captures media and game playback; it does not record microphone input.\n\n" +
                "2. Approve Android's screen-capture prompt. Android requires a new approval for every recording.\n\n" +
                "Keep the phone awake while you approve both prompts. Some apps block playback capture, so a recording can still have silent audio.");
        message.setTextSize(18);
        int padding = (int) (24 * getResources().getDisplayMetrics().density);
        message.setPadding(padding, padding, padding, padding);
        setContentView(message);
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO) != PackageManager.PERMISSION_GRANTED) {
            requestPermissions(new String[]{Manifest.permission.RECORD_AUDIO}, REQUEST_RECORD_AUDIO);
        } else {
            requestProjectionConsent();
        }
    }

    @Override protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        if (ACTION_STOP.equals(intent.getAction())) {
            String requestedId = intent.getStringExtra(EXTRA_SESSION);
            if (sessionId != null && sessionId.equals(requestedId)) {
                stopSession();
                finish();
            }
        }
    }

    @Override public void onRequestPermissionsResult(int requestCode, String[] permissions, int[] grantResults) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults);
        if (requestCode != REQUEST_RECORD_AUDIO) return;
        if (grantResults.length == 0 || grantResults[0] != PackageManager.PERMISSION_GRANTED) {
            status("audio_permission_denied", "Android RECORD_AUDIO permission was denied.");
            finish();
            return;
        }
        requestProjectionConsent();
    }

    private void requestProjectionConsent() {
        if (projectionRequested) return;
        projectionRequested = true;
        MediaProjectionManager manager = getSystemService(MediaProjectionManager.class);
        if (manager == null) {
            status("unavailable", "Android MediaProjection service is unavailable.");
            finish();
            return;
        }
        startActivityForResult(manager.createScreenCaptureIntent(), REQUEST_PROJECTION);
    }

    @Override protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (requestCode != REQUEST_PROJECTION) return;
        if (resultCode != RESULT_OK || data == null) {
            status("projection_consent_denied", "Android screen-capture consent was not granted.");
            finish();
            return;
        }
        Intent service = new Intent(this, CaptureService.class)
                .setAction(CaptureService.ACTION_START)
                .putExtra(EXTRA_SESSION, sessionId)
                .putExtra(CaptureService.EXTRA_RESULT_CODE, resultCode)
                .putExtra(CaptureService.EXTRA_RESULT_DATA, data);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) startForegroundService(service);
        else startService(service);
        // Returning to the previous app keeps the target screen visible while
        // the foreground capture service owns the projection token.
        finish();
    }

    private void stopSession() {
        String requestedId = getIntent().getStringExtra(EXTRA_SESSION);
        if (requestedId != null && SESSION_ID.matcher(requestedId).matches()) {
            Intent stop = new Intent(this, CaptureService.class)
                    .setAction(CaptureService.ACTION_STOP)
                    .putExtra(EXTRA_SESSION, requestedId);
            // This command only stops an existing foreground service. Starting
            // it as an FGS would require another startForeground() call and can
            // trip Android's foreground-service startup deadline.
            startService(stop);
        }
        finish();
    }

    private void status(String state, String detail) {
        if (sessionId == null || !SESSION_ID.matcher(sessionId).matches()) return;
        File musicDirectory = getExternalFilesDir(Environment.DIRECTORY_MUSIC);
        if (musicDirectory == null) return;
        File directory = new File(musicDirectory, "DeviceControlAudio/" + sessionId);
        CaptureService.writeStatus(directory, state, detail, 0L, 0L, 0, 0.0);
    }
}
